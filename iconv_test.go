//
// iconv_test.go
//
package iconv

import (
	"testing"
	"strconv"
)

var testData = []struct{utf8, other, otherEncoding string} {
	{"新浪", "\xd0\xc2\xc0\xcb", "GB2312"},
	{"これは漢字です。", "\x82\xb1\x82\xea\x82\xcd\x8a\xbf\x8e\x9a\x82\xc5\x82\xb7\x81B", "SJIS"},
	{"これは漢字です。", "S0\x8c0o0\"oW[g0Y0\x020", "UTF-16LE"},
	{"これは漢字です。", "0S0\x8c0oo\"[W0g0Y0\x02", "UTF-16BE"},
	{"€1 is cheap", "\xa41 is cheap", "ISO-8859-15"},
	{"", "", "SJIS"},
}

func TestIconv(t *testing.T) {
	for _, data := range testData {
		cd, err := Open("UTF-8", data.otherEncoding)
		if err != nil {
			t.Errorf("Error on opening: %s\n", err)
			continue
		}

		str, err := cd.Conv([]byte(data.other))
		if err != nil {
			t.Errorf("Error on conversion: %s\n", err)
			continue
		}

		if string(str) != data.utf8 {
			t.Errorf("Unexpected value: %#v (expected %#v)", str, data.utf8)
		}

		err = cd.Close()
		if err != nil {
			t.Errorf("Error on close: %s\n", err)
		}
	}
}

func TestIconvReverse(t *testing.T) {
	for _, data := range testData {
		cd, err := Open(data.otherEncoding, "UTF-8")
		if err != nil {
			t.Errorf("Error on opening: %s\n", err)
			continue
		}

		str, err := cd.Conv([]byte(data.utf8))
		if err != nil {
			t.Errorf("Error on conversion: %s\n", err)
			continue
		}

		if string(str) != data.other {
			t.Errorf("Unexpected value: %#v (expected %#v)", str, data.other)
		}

		err = cd.Close()
		if err != nil {
			t.Errorf("Error on close: %s\n", err)
		}
	}
}

func TestInvalidEncoding(t *testing.T) {
	_, err := Open("INVALID_ENCODING", "INVALID_ENCODING")
	if err != InvalidArgument {
		t.Errorf("should've been error")
		return
	}
}

func TestDiscardUnrecognized(t *testing.T) {
	cd, err := OpenWithFallback(testData[1].otherEncoding, "UTF-8", DISCARD_UNRECOGNIZED)
	if err != nil {
		t.Errorf("Error on opening: %s\n", err)
		return
	}
	b, err := cd.Conv([]byte(testData[0].other))
	if len(b) > 0 {
		t.Errorf("should discard all")
	}
	cd.Close()
}

func TestKeepUnrecognized(t *testing.T) {
	cd, err := OpenWithFallback(testData[1].otherEncoding, "UTF-8", KEEP_UNRECOGNIZED)
	if err != nil {
		t.Errorf("Error on opening: %s\n", err)
		return
	}
	b, err := cd.Conv([]byte(testData[0].other))
	if string(b) != testData[0].other {
		t.Errorf("should be the same as the original input")
	}
	cd.Close()
}

func TestMultipleEncodings(t *testing.T) {
	input := testData[0].other + "; " + testData[1].other
	expected := testData[0].utf8 + "; " + testData[1].utf8
	
	cd1, err := Open("UTF-8", testData[0].otherEncoding)
	if err != nil {
		t.Errorf("Error on opening: %s\n", err)
		return
	}
	
	b, err := cd1.Conv([]byte(input))
	if err != nil {
		t.Errorf("Error on conversion: %s\n", err)
		return
	}
	println(strconv.QuoteToASCII(testData[0].utf8 + "; " + testData[1].other))
	println(strconv.QuoteToASCII(string(b)))
	
	input2 := string(b)
	
	cd2, err := Open("UTF-8", testData[0].otherEncoding)
	if err != nil {
		t.Errorf("Error on opening: %s\n", err)
		return
	}
	
	b, err = cd2.Conv([]byte(input2))
	
	if err != nil {
		t.Errorf("Error on conversion: %s\n", err)
		return
	}
	
	println(strconv.QuoteToASCII(expected))
	println(strconv.QuoteToASCII(string(b)))
	
	if string(b) == expected {
		t.Errorf("mix failed")
	}
	cd1.Close()
	cd2.Close()
}

