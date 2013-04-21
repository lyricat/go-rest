package rest

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
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
