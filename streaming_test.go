package rest

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"
)

type FakeStreaming struct {
	last map[string]string
}

func (f FakeStreaming) NoInput(s Stream) {
	f.last["method"] = "NoInput"
	f.last["input"] = ""
}

func (f FakeStreaming) Input(s Stream, input string) {
	f.last["method"] = "Input"
	f.last["input"] = input
}

func (f FakeStreaming) ErrorEmpty() {}

func (f FakeStreaming) ErrorStream(input string) {}

func (f FakeStreaming) ErrorMore(s Stream, input string, other int) {}

func (f FakeStreaming) ErrorReturn(s Stream) string { return "" }

func TestNewStreaming(t *testing.T) {
	type Test struct {
		path pathFormatter
		f    reflect.Method
		tag  reflect.StructTag

		ok        bool
		funcIndex int
		request   string
		end       string
	}
	s := new(FakeStreaming)
	instance := reflect.ValueOf(s).Elem()
	instanceType := instance.Type()
	ni, ok := instanceType.MethodByName("NoInput")
	if !ok {
		t.Fatal("no NoInput")
	}
	i, ok := instanceType.MethodByName("Input")
	if !ok {
		t.Fatal("no Input")
	}
	ee, ok := instanceType.MethodByName("ErrorEmpty")
	if !ok {
		t.Fatal("no ErrorEmpty")
	}
	es, ok := instanceType.MethodByName("ErrorStream")
	if !ok {
		t.Fatal("no ErrorStream")
	}
	em, ok := instanceType.MethodByName("ErrorMore")
	if !ok {
		t.Fatal("no ErrorMore")
	}
	er, ok := instanceType.MethodByName("ErrorReturn")
	if !ok {
		t.Fatal("no ErrorReturn")
	}
	var tests = []Test{
		{"/", ni, `end:"\n"`, true, ni.Index, "<nil>", "\n"},
		{"/", i, "", true, i.Index, "string", ""},
		{"/", ee, "", false, ee.Index, "", ""},
		{"/", es, "", false, es.Index, "", ""},
		{"/", em, "", false, em.Index, "", ""},
		{"/", er, "", false, er.Index, "", ""},
	}
	for i, test := range tests {
		inner := new(innerStreaming)
		err := inner.init(test.path, test.f, test.tag)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			continue
		}
		assert.Equal(t, inner.formatter, test.path, fmt.Sprintf("test %d", i))
		assert.Equal(t, inner.funcIndex, test.funcIndex, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", inner.requestType), test.request, fmt.Sprintf("test %d", i))
		assert.Equal(t, inner.end, test.end, fmt.Sprintf("test %d", i))
	}
}

func TestStreamingHandle(t *testing.T) {
	type Test struct {
		path        string
		f           reflect.Method
		tag         reflect.StructTag
		requestBody string

		code  int
		vars  map[string]string
		input string
	}
	s := new(FakeStreaming)
	s.last = make(map[string]string)
	instance := reflect.ValueOf(s).Elem()
	instanceType := instance.Type()
	ni, ok := instanceType.MethodByName("NoInput")
	if !ok {
		t.Fatal("no NoInput")
	}
	i, ok := instanceType.MethodByName("Input")
	if !ok {
		t.Fatal("no Input")
	}

	var tests = []Test{
		{"/", ni, "", "", http.StatusOK, nil, ""},
		{"/", i, "", "\"input\"", http.StatusOK, nil, "input"},
	}
	for i, test := range tests {
		inner := new(innerStreaming)
		err := inner.init(pathToFormatter("", test.path), test.f, test.tag)
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		buf := bytes.NewBufferString(test.requestBody)
		req, err := http.NewRequest("GET", "http://fake.domain"+test.path, buf)
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		h := newHijacker()
		ctx, err := newContext(h, req, test.vars, "application/json", "utf-8")
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		inner.handle(instance, ctx)
		assert.Equal(t, h.code, http.StatusOK, fmt.Sprintf("test %d code: %d", i, h.code))
		assert.Equal(t, inner.formatter, test.path, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["method"], test.f.Name, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["input"], test.input, fmt.Sprintf("test %d", i))
	}
}

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
