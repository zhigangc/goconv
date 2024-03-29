package goconv
// #cgo CFLAGS: -I/usr/include
// #cgo LDFLAGS: -liconv
// #include <iconv.h>
// #include <errno.h>
// #include <stdlib.h>
import "C"
import (
	"bytes"
	"os"
	"io"
	"unsafe"
)

type Iconv struct {
	pIconv C.iconv_t
	fallbackPolicy int
	fallback func([]byte, io.Writer, []byte) (int, os.Error)
	fallbackIconv *Iconv
}

const (
	DISCARD_UNRECOGNIZED = 0
	KEEP_UNRECOGNIZED = 1
	NEXT_ENC_UNRECOGNIZED = 2
)

var (
	NilIconvPointer = os.NewError("Nil iconv pointer")
	NilFallbackIconv = os.NewError("Nil fallback iconv")
	InvalidFallbackPolicy = os.NewError("Invalid fallback policy")
	FallbackCannotAdvance = os.NewError("Fallback cannot advance conversion")
	InvalidSequence = os.Errno(int(C.EILSEQ))
	OutputBufferInsufficient = os.Errno(int(C.E2BIG))
	IncompleteSequence = os.Errno(int(C.EINVAL))
	InvalidArgument = os.Errno(int(C.EINVAL))
)

func fallbackDiscardUnrecognized(input []byte, out io.Writer, outBuf []byte) (bytesConverted int, err os.Error) {
	bytesConverted = len(input)
	return
}

func fallbackKeepIntactUnrecognized(input []byte, out io.Writer, outBuf []byte) (bytesConverted int, err os.Error) {
	out.Write(input)
	bytesConverted = len(input)
	return
}

func OpenWithFallback(fromCode string, toCode string, fallbackPolicy int) (ic *Iconv, err os.Error) {
	var pIconv C.iconv_t
	
	toCodeCharPtr := C.CString(toCode)
	defer C.free(unsafe.Pointer(toCodeCharPtr))
	fromCodeCharPtr := C.CString(fromCode)
	defer C.free(unsafe.Pointer(fromCodeCharPtr))

	pIconv, err = C.iconv_open(toCodeCharPtr, fromCodeCharPtr)
	if err == nil {
		if pIconv == nil {
			err = NilIconvPointer
			return
		}
		if fallbackPolicy == DISCARD_UNRECOGNIZED {
			ic = &Iconv{pIconv: pIconv, fallbackPolicy: fallbackPolicy, fallback: fallbackDiscardUnrecognized}
		} else if fallbackPolicy == KEEP_UNRECOGNIZED {
			ic = &Iconv{pIconv: pIconv, fallbackPolicy: fallbackPolicy, fallback: fallbackKeepIntactUnrecognized}
		} else if fallbackPolicy == NEXT_ENC_UNRECOGNIZED {
			ic = &Iconv{pIconv: pIconv, fallbackPolicy: fallbackPolicy}
		} else {
			err = InvalidFallbackPolicy
		}
	}
	return
}

func (ic *Iconv) SetFallback(fallbackIconv *Iconv) {
	ic.fallbackIconv = fallbackIconv
	return
}

func Open(fromCode string, toCode string) (ic *Iconv, err os.Error) {
	ic, err = OpenWithFallback(fromCode, toCode, DISCARD_UNRECOGNIZED)
	return
}

func (ic *Iconv) Close() (err os.Error) {
	_, err = C.iconv_close(ic.pIconv)
	if ic.fallbackIconv != nil {
		ic.fallbackIconv.Close()
		ic.fallbackIconv = nil
	}
	return
}

func (ic *Iconv) convert(input []byte, out io.Writer, outBuf []byte) (bytesConverted int, err os.Error) {
	inputLen := len(input)
	if inputLen == 0 {
		return
	}

	outputLen := len(outBuf)
	if outputLen == 0 {
		outputLen = inputLen
		outBuf = make([]byte, outputLen)
	}

	outputPtr := &outBuf[0]
	outputPtrPtr := (**C.char)(unsafe.Pointer(&outputPtr))
	outputBytesLeft := C.size_t(outputLen)
	
	inputPtr := &input[0]
	inputPtrPtr := (**C.char)(unsafe.Pointer(&inputPtr))
	inputBytesLeft := C.size_t(inputLen)
		
	_, err = C.iconv(ic.pIconv, inputPtrPtr, &inputBytesLeft, outputPtrPtr, &outputBytesLeft)
	bytesConverted = inputLen - int(inputBytesLeft)
	if int(outputBytesLeft) < outputLen {
		out.Write(outBuf[:outputLen-int(outputBytesLeft)])
	}
	return
}

//err returns the last error
func (ic *Iconv) Conv(input []byte) (output []byte, err os.Error) {
	inputLen := len(input)
	if inputLen == 0 {
		output = input
		return
	}
	buf := &bytes.Buffer{}
	outBuf := make([]byte, inputLen)
	offset := 0
	bytesConverted := 0
	
	for offset < inputLen {
		bytesConverted, err = ic.convert(input[offset:], buf, outBuf)
		offset += bytesConverted
		bytesConverted = 0
		if err == InvalidSequence || err == IncompleteSequence {
			if ic.fallbackPolicy == NEXT_ENC_UNRECOGNIZED {
				if ic.fallbackIconv == nil {
					err = NilFallbackIconv
					return
				}
				bytesConverted, err = ic.fallbackIconv.convert(input[offset:], buf, outBuf)
			} else {
				bytesConverted, err = ic.fallback(input[offset:offset+1], buf, outBuf)
			}
			if bytesConverted > 0 {
				offset += bytesConverted
			} else {
				err = FallbackCannotAdvance
				break
			}
		}
	}
	output = buf.Bytes()
	return
}