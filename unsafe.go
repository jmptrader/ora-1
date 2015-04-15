package ora

import (
	"log"
	"unsafe"
)

const sizeOfInt = int(unsafe.Sizeof(int(0)))

func logLine(args ...interface{}) {
	log.Println(args...)
}

func intRef(p *int) uintptr {
	return uintptr(unsafe.Pointer(p))
}

func float64Ref(p *float64) uintptr {
	return uintptr(unsafe.Pointer(p))
}

func int64Ref(p *int64) uintptr {
	return uintptr(unsafe.Pointer(p))
}

func bufAddr(p []byte) uintptr {
	if len(p) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&p[0]))
}

func bufRef(p *[]byte) uintptr {
	return uintptr(unsafe.Pointer(p))
}

func ref(p *uintptr) uintptr {
	return uintptr(unsafe.Pointer(p))
}
