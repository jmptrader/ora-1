package ora

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"
)

type connStruct struct {
	env    *ociHandle
	serv   *ociHandle
	err    *ociHandle
	tx     *Transaction
	opened bool
}

// http://docs.oracle.com/cd/B28359_01/appdev.111/b28395/oci16rel001.htm#LNOCI7016
func newConnection() (*connStruct, error) {
	conn := new(connStruct)
	conn.env = &ociHandle{typ: OCI_HTYPE_ENV}

	// TODO: OCI_THREADED
	err := conn.envErr(oci_OCIEnvCreate.Call(conn.env.ref(), OCI_DEFAULT, 0, 0, 0, 0, 0, 0))
	if err != nil {
		return nil, err
	}

	if conn.serv, err = conn.alloc(OCI_HTYPE_SVCCTX); err != nil {
		return nil, err
	}

	if conn.err, err = conn.alloc(OCI_HTYPE_ERROR); err != nil {
		return nil, err
	}

	return conn, nil
}

func (conn *connStruct) Begin() (driver.Tx, error) {
	conn.tx = &Transaction{conn}
	return conn.tx, nil
}

func (conn *connStruct) Prepare(query string) (driver.Stmt, error) {
	stmt, err := conn.newStatement()
	if err != nil {
		return nil, err
	}
	if err = stmt.prepare(query); err != nil {
		return nil, err
	}
	return stmt, nil
}

// TODO: test that connection is actually closed!
func (conn *connStruct) Close() error {
	if conn.opened {
		oci_OCILogoff.Call(conn.serv.ptr, conn.err.ptr)
		conn.opened = false
	}
	oci_OCIHandleFree.Call(conn.serv.ptr, uintptr(OCI_HTYPE_SVCCTX))
	oci_OCIHandleFree.Call(conn.env.ptr, uintptr(OCI_HTYPE_ENV))
	return nil
}

// function for creating statement
func (conn *connStruct) newStatement() (stmt *Statement, err error) {
	stmt = &Statement{conn: conn, tx: conn.tx}
	stmt.ociHandle, err = conn.alloc(OCI_HTYPE_STMT) // allocate prepare statement, later we will need to free it
	return
}

func (conn *connStruct) logon(user, pass, host []byte) (err error) {
	userLen := uintptr(len(user))
	passLen := uintptr(len(pass))
	hostLen := uintptr(len(host))

	if err = conn.cerr(oci_OCILogon.Call(conn.env.ptr, conn.err.ptr, ref(&conn.serv.ptr), bufAddr(user), userLen, bufAddr(pass), passLen, bufAddr(host), hostLen)); err != nil {
		conn.Close()
	} else {
		conn.opened = true
	}
	return
}

func (conn *connStruct) alloc(typ int) (*ociHandle, error) {
	h := &ociHandle{typ: typ}
	if err := conn.envErr(oci_OCIHandleAlloc.Call(conn.env.ptr, h.ref(), uintptr(typ), 0, 0)); err != nil {
		return nil, err
	}
	return h, nil
}

/* for later use
func (conn *connStruct) alloc_descr() *ociHandle {
	h := new(ociHandle)
	err := conn.envErr(oci_OCIDescriptorAlloc.Call(conn.env.ptr, h.ref(), OCI_DTYPE_ROWID, 0, 0))
	if err != nil {
		panic(err)
	}
	return h
}
*/

// function for handling errors from OCI calls
func (conn *connStruct) cerr(r uintptr, r2 uintptr, err error) error {
	return conn.onOCIReturn(int16(r), OCI_HTYPE_ERROR)
}

// function for handling errors on env create and alloc
func (conn *connStruct) envErr(r uintptr, r2 uintptr, err error) error {
	return conn.onOCIReturn(int16(r), OCI_HTYPE_ENV)
}

// http://docs.oracle.com/cd/E11882_01/appdev.112/e10646/oci17msc007.htm#LNOCI17287
func (conn *connStruct) onOCIReturn(code int16, htyp int) error {
	switch code {
	case OCI_SUCCESS:
		return nil
	case OCI_ERROR:
		return conn.getErr(htyp)
	case OCI_INVALID_HANDLE:
		return errors.New("OCI call returned OCI_INVALID_HANDLE")
	}

	return fmt.Errorf("OCI call returned - %d", code)
}

// https://docs.oracle.com/database/121/LNOCI/oci17msc007.htm#LNOCI17287
func (conn *connStruct) getErr(htyp int) error {
	buf := make([]byte, 3072) // OCI_ERROR_MAXMSG_SIZE2 3072
	errcode := 0
	if htyp == OCI_HTYPE_ERROR {
		if err := conn.cerr(oci_OCIErrorGet.Call(conn.err.ptr, uintptr(1), uintptr(0), intRef(&errcode), bufAddr(buf), uintptr(len(buf)), OCI_HTYPE_ERROR)); err != nil {
			return err
		}
	} else {
		if err := conn.cerr(oci_OCIErrorGet.Call(conn.env.ptr, uintptr(1), uintptr(0), intRef(&errcode), bufAddr(buf), uintptr(len(buf)), OCI_HTYPE_ENV)); err != nil {
			return err
		}
	}

	return errors.New(string(buf[:bytes.IndexByte(buf, 0)]))
}
