package iconv
// #cgo CFLAGS: -I/usr/include
// #cgo LDFLAGS: -liconv
// #include <iconv.h>
// #include <errno.h>
// #include <stdlib.h>
import "C"
import (
	"bytes"
	"os"
	"unsafe"
)

type Iconv struct {
	pIconv C.iconv_t
}

var (
	NilIconv = os.NewError("Nil iconv object")
	InvalidSequence = os.Errno(int(C.EILSEQ))
	OutputBufferInsufficient = os.Errno(int(C.E2BIG))
	IncompleteSequence = os.Errno(int(C.EINVAL))
	InvalidArgument = os.Errno(int(C.EINVAL))
)

func Open(toCode string, fromCode string) (ic *Iconv, err os.Error) {
	var pIconv C.iconv_t
	
	toCodeCharPtr := C.CString(toCode)
	defer C.free(unsafe.Pointer(toCodeCharPtr))
	fromCodeCharPtr := C.CString(fromCode)
	defer C.free(unsafe.Pointer(fromCodeCharPtr))

	pIconv, err = C.iconv_open(toCodeCharPtr, fromCodeCharPtr)
	if err == nil {
		if pIconv == nil {
			err = NilIconv
		}
		ic = &Iconv{pIconv: pIconv}
	} else if err == InvalidArgument {
		err = NilIconv
	}
	return
}

func (ic *Iconv) Close() (err os.Error) {
	_, err = C.iconv_close(ic.pIconv)
	return
}

//err returns the last error
func (ic *Iconv) Conv(input []byte) (output []byte, err os.Error) {
	totalInputLen := len(input)
	if totalInputLen == 0 {
		output = input
		return
	}
	var buf bytes.Buffer
	outputLimit := totalInputLen	
	output = make([]byte, outputLimit)
	outputPtr := &output[0]
	outputPtrPtr := (**C.char)(unsafe.Pointer(&outputPtr))
	outputBytes := C.size_t(outputLimit)
	
	offset := 0
	inputPtr := &input[offset]
	inputLen := len(input[offset:])
	inputPtrPtr := (**C.char)(unsafe.Pointer(&inputPtr))
	inputBytes := C.size_t(inputLen)
		
	for inputBytes > 0 && offset < totalInputLen {
		_, err = C.iconv(ic.pIconv, inputPtrPtr, &inputBytes, outputPtrPtr, &outputBytes)
		if int(outputBytes) < outputLimit {
			buf.Write(output[:outputLimit-int(outputBytes)])
			outputPtr = &output[0]
			outputPtrPtr = (**C.char)(unsafe.Pointer(&outputPtr))
			outputBytes = C.size_t(outputLimit)
		}

		if err == InvalidSequence || err == IncompleteSequence {
			offset += (inputLen - int(inputBytes))
			buf.WriteByte(input[offset])
			offset += 1
			if offset < totalInputLen {
				inputPtr = &input[offset]
				inputLen = len(input[offset:])
				inputPtrPtr = (**C.char)(unsafe.Pointer(&inputPtr))
				inputBytes = C.size_t(inputLen)
			}
		}
	}
	output = buf.Bytes()
	return
}