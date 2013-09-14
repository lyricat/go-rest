package rest

import (
	"bytes"
	"fmt"
	"github.com/googollee/go-assert"
	"testing"
)

type FakeWriter struct {
	ok  bool
	max int
	buf *bytes.Buffer
}

func (w *FakeWriter) Write(p []byte) (int, error) {
	if !w.ok {
		return -1, fmt.Errorf("error")
	}
	ret := w.max
	if len(p) < ret {
		ret = len(p)
	}
	if w.buf == nil {
		w.buf = bytes.NewBuffer(nil)
	}
	_, err := w.buf.Write(p[:ret])
	return ret, err
}

func TestErrorWriter(t *testing.T) {
	innerWriter := new(FakeWriter)
	innerWriter.ok = true
	innerWriter.max = 10
	w := NewErrorWriter(innerWriter)

	w.Write([]byte("12345"))
	assert.Equal(t, w.Error(), nil)
	assert.Equal(t, innerWriter.buf.String(), "12345")

	w.Write([]byte("12345678901234567890"))
	assert.Equal(t, w.Error(), nil)
	assert.Equal(t, innerWriter.buf.String(), "1234512345678901234567890")

	innerWriter.ok = false
	w.Write([]byte("12345"))
	err := w.Error()
	assert.NotEqual(t, err, nil)
	assert.Equal(t, innerWriter.buf.String(), "1234512345678901234567890")

	w.Write([]byte("12345"))
	assert.Equal(t, w.Error(), err)
	assert.Equal(t, innerWriter.buf.String(), "1234512345678901234567890")

	innerWriter.ok = true
	w.Write([]byte("12345"))
	assert.Equal(t, w.Error(), err)
	assert.Equal(t, innerWriter.buf.String(), "1234512345678901234567890")
}
