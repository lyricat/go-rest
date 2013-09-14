package rest

import (
	"bytes"
	"fmt"
	"github.com/googollee/go-assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type streamFuncs struct {
	ctx    StreamContext
	called string
}

func (f *streamFuncs) Ctx(ctx StreamContext) {
	f.ctx = ctx
	f.called = "called ctx"
}

func (f *streamFuncs) CtxInt(ctx StreamContext, i int) {
	f.ctx = ctx
	f.called = fmt.Sprintf("called ctxInt with %d", i)
}

func (f *streamFuncs) CtxPString(ctx StreamContext, ps *string) {
	f.ctx = ctx
	f.called = fmt.Sprintf("called ctxPString with %s", *ps)
}

func (f *streamFuncs) NoArg()                              {}
func (f *streamFuncs) WithReturn(ctx StreamContext) int    { return 1 }
func (f *streamFuncs) NoStreamCtx(ctx Context)             {}
func (f *streamFuncs) MoreArg(ctx StreamContext, i, j int) {}
func (f *streamFuncs) NoContext1(i int)                    {}
func (f *streamFuncs) NoContext2(i, j int)                 {}

func TestStream(t *testing.T) {
	type Test struct {
		serviceTag reflect.StructTag
		fieldTag   reflect.StructTag
		fname      string
		f          reflect.Value

		ok         bool
		path       string
		method     string
		marshaller string
		inputType  string
	}
	f := new(streamFuncs)
	var tests = []Test{
		{`mime:"application/json"`, `method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{`mime:"application/json" prefix:"/prefix"`, `method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/prefix", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{`prefix:"/prefix"`, `route:"/route" method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/prefix/route", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{`prefix:"/prefix"`, `path:"/path" method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/path", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{``, `method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{``, `method:"GET"`, "CtxInt", reflect.ValueOf(f.CtxInt), true, "/", "GET", "rest.JSONMarshaller{}", "int"},
		{``, `method:"GET"`, "CtxPString", reflect.ValueOf(f.CtxPString), true, "/", "GET", "rest.JSONMarshaller{}", "*string"},

		{``, ``, "NoMethod", reflect.ValueOf(f.Ctx), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoArg", reflect.ValueOf(f.NoArg), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "WithReturn", reflect.ValueOf(f.WithReturn), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoStreamCtx", reflect.ValueOf(f.NoStreamCtx), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "MoreArg", reflect.ValueOf(f.MoreArg), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoContext1", reflect.ValueOf(f.NoContext1), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoContext2", reflect.ValueOf(f.NoContext2), false, "/", "", "<nil>", "<nil>"},
	}
	for i, test := range tests {
		var p Node
		p = Streaming{}
		path, method, handler, err := p.CreateHandler(test.serviceTag, test.fieldTag, test.fname, test.f)
		assert.MustEqual(t, err == nil, test.ok, "test %d: %s", i, err)
		if err != nil {
			continue
		}
		assert.Equal(t, path, test.path, "test %d", i)
		assert.Equal(t, method, test.method, "test %d", i)
		ph, ok := handler.(*streamHandler)
		assert.MustEqual(t, ok, true, "test %d", i)
		assert.Equal(t, ph.name, test.fname, "test %d", i)
		assert.Equal(t, fmt.Sprintf("%#v", ph.marshaller), test.marshaller, "test %d", i)
		assert.Equal(t, fmt.Sprintf("%v", ph.inputType), test.inputType, "test %d", i)
	}
}

type FakeStreamFuncHandler struct {
	handler *streamHandler
	vars    map[string]string
}

func (h *FakeStreamFuncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r, h.vars)
}

func TestStreamHandler(t *testing.T) {
	type Test struct {
		name       string
		marshaller Marshaller
		inputType  reflect.Type
		f          reflect.Value
		header     map[string][]string
		body       string
		vars       map[string]string

		called           string
		targetMarshaller Marshaller
	}
	f := new(streamFuncs)
	fakeMarshaller := FakeMarshaller{}
	RegisterMarshaller("fake/mime", fakeMarshaller)
	str := ""
	var tests = []Test{
		{"Ctx", fakeMarshaller, reflect.TypeOf(nil), reflect.ValueOf(f.Ctx), map[string][]string{"Content-Type": []string{"application/json"}}, "", nil, "called ctx", jsonMarshaller},
		{"Ctx", jsonMarshaller, reflect.TypeOf(nil), reflect.ValueOf(f.Ctx), nil, "", map[string]string{"ab": "cd"}, "called ctx", jsonMarshaller},
		{"CtxInt", jsonMarshaller, reflect.TypeOf(1), reflect.ValueOf(f.CtxInt), nil, "1", nil, "called ctxInt with 1", jsonMarshaller},
		{"CtxPString", jsonMarshaller, reflect.TypeOf(&str), reflect.ValueOf(f.CtxPString), nil, `"str"`, nil, "called ctxPString with str", jsonMarshaller},
	}
	for i, test := range tests {
		handler := &streamHandler{
			name:       test.name,
			marshaller: test.marshaller,
			inputType:  test.inputType,
			f:          test.f,
		}
		h := &FakeStreamFuncHandler{
			handler: handler,
			vars:    test.vars,
		}
		server := httptest.NewServer(h)
		req, err := http.NewRequest("GET", server.URL, strings.NewReader(test.body))
		assert.MustEqual(t, err, nil, "test %d", i)
		req.Header = test.header
		resp, err := http.DefaultClient.Do(req)
		assert.MustEqual(t, err, nil, "test %d", i)
		assert.Equal(t, resp.StatusCode, http.StatusOK, "test %d", i)
		assert.Equal(t, f.called, test.called, "test %d", i)
		ctx, ok := f.ctx.(*streamContext)
		assert.MustEqual(t, ok, true, "test %d", i)
		assert.Equal(t, ctx.handlerName, test.name, "test %d", i)
		assert.Equal(t, ctx.vars, test.vars, "test %d", i)
		assert.Equal(t, ctx.marshaller, test.targetMarshaller, "test %d", i)
		server.Close()
	}
}

func TestStreamHandlerHijackFailed(t *testing.T) {
	f := new(streamFuncs)
	handler := &streamHandler{
		name:       "name",
		marshaller: jsonMarshaller,
		inputType:  nil,
		f:          reflect.ValueOf(f.Ctx),
	}
	req, err := http.NewRequest("GET", "http://domain/path", nil)
	assert.MustEqual(t, err, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req, nil)
	assert.Equal(t, resp.Code, http.StatusInternalServerError)
}

func TestStreamHandlerUnmarshallFailed(t *testing.T) {
	RegisterMarshaller("mime/fail", new(FailMarshaller))
	f := new(streamFuncs)
	handler := &streamHandler{
		name:       "name",
		marshaller: jsonMarshaller,
		inputType:  reflect.TypeOf(1),
		f:          reflect.ValueOf(f.Ctx),
	}
	h := &FakeStreamFuncHandler{
		handler: handler,
		vars:    nil,
	}
	server := httptest.NewServer(h)
	defer server.Close()
	req, err := http.NewRequest("GET", server.URL, strings.NewReader("1"))
	assert.MustEqual(t, err, nil)
	req.Header.Set("Content-Type", "mime/fail")
	resp, err := http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func TestStreamResponseWriter(t *testing.T) {
	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.WriteHeader(http.StatusOK)
		assert.Equal(t, buf.String(), "HTTP/1.1 200 OK\r\n\r\n")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.Write([]byte("response body"))
		assert.Equal(t, buf.String(), "HTTP/1.1 200 OK\r\n\r\nresponse body")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("response body"))
		assert.Equal(t, buf.String(), "HTTP/1.1 400 Bad Request\r\n\r\nresponse body")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.Header().Set("Custom-Header", "value")
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("response body"))
		assert.Equal(t, buf.String(), "HTTP/1.1 400 Bad Request\r\nCustom-Header: value\r\n\r\nresponse body")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.Header().Set("Custom-Header", "value")
		resp.WriteHeader(http.StatusBadRequest)
		resp.Header().Set("Should-Not-Include", "nonexist")
		resp.Write([]byte("response body"))
		assert.Equal(t, buf.String(), "HTTP/1.1 400 Bad Request\r\nCustom-Header: value\r\n\r\nresponse body")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.Header().Set("Custom-Header", "value")
		resp.WriteHeader(http.StatusBadRequest)
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte("response body"))
		assert.Equal(t, buf.String(), "HTTP/1.1 400 Bad Request\r\nCustom-Header: value\r\n\r\nresponse body")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.Header().Set("Custom-Header", "value")
		resp.Write([]byte("response body"))
		resp.Header().Set("Should-Not-Include", "nonexist")
		assert.Equal(t, buf.String(), "HTTP/1.1 200 OK\r\nCustom-Header: value\r\n\r\nresponse body")
	}

	{
		buf := bytes.NewBuffer(nil)
		resp := newStreamResponseWriter(buf)
		resp.Header().Set("Custom-Header", "value")
		resp.Write([]byte("response body"))
		resp.WriteHeader(http.StatusBadRequest)
		assert.Equal(t, buf.String(), "HTTP/1.1 200 OK\r\nCustom-Header: value\r\n\r\nresponse body")
	}
}

type FakeStreamHandler struct {
	t *testing.T
	p chan int
	r chan int
}

func (h *FakeStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, err := newStreamContext("context", jsonMarshaller, "utf-8", nil, "\nend\n", r, w)
	assert.MustEqual(h.t, err, nil)
	defer ctx.close()

	_, ok := ctx.Response().(*streamResponseWriter)
	assert.MustEqual(h.t, ok, true)
	ctx.Return(http.StatusOK, "with resp %d", http.StatusOK)
	h.p <- 1
	<-h.r
	assert.Equal(h.t, ctx.Ping(), nil)
	assert.Equal(h.t, ctx.SetWriteDeadline(time.Now().Add(time.Second/10)), nil)
	assert.Equal(h.t, ctx.Render(1), nil)
	h.p <- 1
	<-h.r
	assert.Equal(h.t, ctx.SetWriteDeadline(time.Now().Add(time.Second/10)), nil)
	assert.Equal(h.t, ctx.Render(2), nil)
	time.Sleep(time.Second / 10)
	assert.NotEqual(h.t, ctx.Render(3), nil)
	assert.Equal(h.t, ctx.Ping(), nil)
	h.p <- 1
	<-h.r
	assert.NotEqual(h.t, ctx.Ping(), nil)
	h.p <- 1
}

func TestStreamContext(t *testing.T) {
	p, r := make(chan int), make(chan int)
	server := httptest.NewServer(&FakeStreamHandler{t, p, r})
	defer server.Close()
	req, err := http.NewRequest("GET", server.URL, nil)
	assert.MustEqual(t, err, nil)
	resp, err := http.DefaultClient.Do(req)
	<-p
	assert.MustEqual(t, err, nil, "error: %s", err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	r <- 1
	<-p
	b := make([]byte, 1024)
	n, err := resp.Body.Read(b[:])
	assert.MustEqual(t, err, nil, "error: %s", err)
	assert.Equal(t, string(b[:n]), "with resp 200\n")
	n, err = resp.Body.Read(b[:])
	assert.MustEqual(t, err, nil, "error: %s", err)
	assert.Equal(t, string(b[:n]), "1\n\nend\n")
	r <- 1
	<-p
	n, err = resp.Body.Read(b[:])
	assert.MustEqual(t, err, nil, "error: %s", err)
	assert.Equal(t, string(b[:n]), "2\n\nend\n")
	resp.Body.Close()
	r <- 1
	<-p
}

func TestStreamHandlerFail(t *testing.T) {
	failMarshaller := FailMarshaller{}
	RegisterMarshaller("fail/mime", failMarshaller)

	{
		req, err := http.NewRequest("GET", "http://method", strings.NewReader("1"))
		assert.MustEqual(t, err, nil)
		req.Header.Set("Content-Type", "fail/mime")
		resp := httptest.NewRecorder()
		resp.Code = http.StatusOK
		_, err = newStreamContext("context", failMarshaller, "utf-8", nil, "\nend\n", req, resp)
		assert.NotEqual(t, err, nil)
	}

	{
		req, err := http.NewRequest("GET", "http://method", strings.NewReader("1"))
		assert.MustEqual(t, err, nil)
		req.Header.Set("Content-Type", "fail/mime")
		resp := httptest.NewRecorder()
		resp.Code = http.StatusOK
		_, err = newStreamContext("context", failMarshaller, "utf-8", nil, "\nend\n", req, resp)
		base := newBaseContext("context", failMarshaller, "utf-8", nil, req, resp)
		ctx := &streamContext{
			baseContext: base,
		}
		err = ctx.Render(1)
		assert.NotEqual(t, err, nil)
	}
}
