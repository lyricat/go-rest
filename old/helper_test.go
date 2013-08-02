package rest

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

type fakeAddr string

func (a fakeAddr) Network() string {
	return string(a)
}

func (a fakeAddr) String() string {
	return string(a)
}

type fakeConn struct {
	buf *bytes.Buffer
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		buf: bytes.NewBuffer(nil),
	}
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	return c.buf.Read(b)
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
	return c.buf.Write(b)
}

func (c *fakeConn) Close() error {
	return nil
}

func (c *fakeConn) LocalAddr() net.Addr {
	return fakeAddr("")
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return fakeAddr("")
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type fakeHijacker struct {
	code   int
	header http.Header
	conn   *fakeConn
}

func newHijacker() *fakeHijacker {
	return &fakeHijacker{
		code:   http.StatusOK,
		header: make(http.Header),
		conn:   newFakeConn(),
	}
}

func (w *fakeHijacker) Header() http.Header {
	return w.header
}

func (w *fakeHijacker) Write(p []byte) (int, error) {
	return w.conn.Write(p)
}

func (w *fakeHijacker) WriteHeader(code int) {
	w.code = code
}

func (w *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	bufrw := bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn))
	return w.conn, bufrw, nil
}

func getWhitespaceString() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	parts := strings.Split(file, "/")
	file = parts[len(parts)-1]

	return strings.Repeat(" ", len(fmt.Sprintf("%s:%d:      ", file, line)))
}

func callerInfo() string {
	file := ""
	line := 0
	ok := false

	for i := 0; ; i++ {
		_, file, line, ok = runtime.Caller(i)
		if !ok {
			return ""
		}
		parts := strings.Split(file, "/")
		dir := parts[len(parts)-2]
		file = parts[len(parts)-1]
		if (dir != "assert" && dir != "mock" && file != "helper_test.go") || file == "mock_test.go" {
			break
		}
	}

	return fmt.Sprintf("%s:%d", file, line)
}

func equal(t *testing.T, a, b interface{}, args ...interface{}) {
	msg := ""
	if len(args) == 1 {
		msg = args[0].(string)
	}
	if len(args) > 1 {
		msg = fmt.Sprintf(args[0].(string), args[1:]...)
	}

	if reflect.DeepEqual(a, b) {
		return
	}
	if reflect.ValueOf(a) == reflect.ValueOf(b) {
		return
	}
	if fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b) {
		return
	}

	if len(msg) > 0 {
		t.Errorf("\r%s\r\tLocation:\t%s\n\r\tError:\t\tNot equal: %#v != %#v\n\r\tMessages:\t%s\n\r", getWhitespaceString(), callerInfo(), a, b, msg)
	} else {
		t.Errorf("\r%s\r\tLocation:\t%s\n\r\tError:\t\tNot equal: %#v != %#v\n\r", getWhitespaceString(), callerInfo(), a, b)
	}

}
