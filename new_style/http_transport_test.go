package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/googollee/go-assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestParseHeaderField(t *testing.T) {
	type Test struct {
		header string
		field  string
		ret    string
		pair   map[string]string
	}
	var tests = []Test{
		{"", "Abc", "", nil},
		{"text/plain", "Accept", "text/plain", nil},
		{"text/plain; charset=utf8", "Content-Type", "text/plain", map[string]string{"charset": "utf8"}},
		{"text/plain; charset=utf8;", "Content-Type", "text/plain", map[string]string{"charset": "utf8"}},
		{"text/plain; charset", "Content-Type", "text/plain", map[string]string{"charset": ""}},
		{"text/plain; charset=utf8; skin=new", "Content-Type", "text/plain", map[string]string{"charset": "utf8", "skin": "new"}},
	}

	for i, test := range tests {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal("invalid request")
		}
		req.Header.Set(test.field, test.header)
		ret, pair := parseHeaderField(req, test.field)
		assert.Equal(t, ret, test.ret, "test %d", i)
		assert.Equal(t, pair, test.pair, "test %d", i)
	}
}

func TestMergeQuery(t *testing.T) {
	type Test struct {
		query  url.Values
		vars   map[string]string
		expect url.Values
	}
	var tests = []Test{
		{url.Values{"a": []string{"a"}}, map[string]string{"a": "b"}, url.Values{"a": []string{"a", "b"}}},
		{url.Values{}, map[string]string{"a": "b"}, url.Values{"a": []string{"b"}}},
		{url.Values{"a": []string{"a"}}, map[string]string{}, url.Values{"a": []string{"a"}}},
	}
	for i, test := range tests {
		got := mergeQuery(test.query, test.vars)
		assert.Equal(t, got, test.expect, "test %d", i)
	}
}

func TestNormalizePath(t *testing.T) {
	type Test struct {
		prefix string
		expect string
	}
	var tests = []Test{
		{"", "/"},
		{"/", "/"},
		{"/prefix", "/prefix"},
		{"/prefix/", "/prefix"},
		{"prefix/", "/prefix"},
	}
	for i, test := range tests {
		got := normalizePath(test.prefix)
		assert.Equal(t, got, test.expect, "test %d", i)
	}
}

func TestHTTPContext(t *testing.T) {
	type Test struct {
		header             http.Header
		defaultMime        string
		requestOk          bool
		requestMime        string
		requestMarshaller  Marshaller
		responseOk         bool
		responseMime       string
		responseMarshaller Marshaller
	}
	am, bm, cm := TestMarshaller{}, TestMarshaller{}, TestMarshaller{}
	RegisterMarshaller("encode/a", am)
	RegisterMarshaller("encode/b", bm)
	RegisterMarshaller("encode/c", cm)
	var tests = []Test{
		{http.Header{}, "encode/a", true, "encode/a", am, true, "encode/a", am},
		{http.Header{"Content-Type": []string{"encode/b"}}, "encode/a", true, "encode/b", bm, true, "encode/b", bm},
		{http.Header{"Accept": []string{"encode/b"}}, "encode/a", true, "encode/a", am, true, "encode/b", bm},
		{http.Header{"Accept": []string{"encode/b"}, "Content-Type": []string{"encode/c"}}, "encode/a", true, "encode/c", cm, true, "encode/b", bm},

		{http.Header{}, "encode/none", false, "", nil, false, "", nil},
	}
	for i, test := range tests {
		record := httptest.NewRecorder()
		ctx := &HTTPContext{
			r: &http.Request{
				Header: test.header,
			},
			w:           record,
			defaultMime: test.defaultMime,
		}
		err := ctx.parseRequestMarshaller()
		assert.MustEqual(t, err == nil, test.requestOk, "test %d", i)
		assert.Equal(t, ctx.reqMime, test.requestMime, "test %d", i)
		assert.Equal(t, ctx.reqMarshaller, test.requestMarshaller, "test %d", i)
		err = ctx.parseResponseMarshaller()
		assert.MustEqual(t, err == nil, test.responseOk, "test %d", i)
		assert.Equal(t, ctx.respMime, test.responseMime, "test %d", i)
		assert.Equal(t, ctx.respMarshaller, test.responseMarshaller, "test %d", i)

		err = ctx.parseRequestMarshaller()
		assert.MustEqual(t, err == nil, test.requestOk, "test %d", i)
		assert.Equal(t, ctx.reqMime, test.requestMime, "test %d", i)
		assert.Equal(t, ctx.reqMarshaller, test.requestMarshaller, "test %d", i)
		err = ctx.parseResponseMarshaller()
		assert.MustEqual(t, err == nil, test.responseOk, "test %d", i)
		assert.Equal(t, ctx.respMime, test.responseMime, "test %d", i)
		assert.Equal(t, ctx.respMarshaller, test.responseMarshaller, "test %d", i)
	}
}

type TestMarshaller struct{}

func (j TestMarshaller) Marshal(w io.Writer, name string, v interface{}) error { return nil }
func (j TestMarshaller) Unmarshal(r io.Reader, v interface{}) error            { return nil }

func TestContextHandleError(t *testing.T) {
	am, bm, cm := TestMarshaller{}, TestMarshaller{}, TestMarshaller{}
	RegisterMarshaller("encode/a", am)
	RegisterMarshaller("encode/b", bm)
	RegisterMarshaller("encode/c", cm)
	type Test struct {
		header    http.Header
		err       error
		code      int
		errString string
	}
	var tests = []Test{
		{http.Header{}, NewStandardError(1, "abc"), 1, "abc\n"},
		{http.Header{}, NewStandardError(1, 1), 1, ""},
		{http.Header{}, fmt.Errorf("123"), http.StatusInternalServerError, "123\n"},
	}
	for i, test := range tests {
		record := httptest.NewRecorder()
		ctx := &HTTPContext{
			r: &http.Request{
				Header: test.header,
			},
			node:        &namedNode{},
			w:           record,
			defaultMime: "encode/c",
		}
		ctx.handleError(test.err)
		assert.Equal(t, record.Code, test.code, "test %d", i)
		assert.Equal(t, record.Body.String(), test.errString, "test %d", i)
	}
}

func TestStreamingContextHandleError(t *testing.T) {
	type Test struct {
		err  error
		code int
	}
	var tests = []Test{
		{NewStandardError(1, "abc"), 1},
		{fmt.Errorf("123"), http.StatusInternalServerError},
	}
	ctx := new(HTTPStreamingContext)
	for i, test := range tests {
		ctx.handleError(test.err)
		assert.Equal(t, ctx.code, test.code, "test %d", i)
	}
}

type Succeed struct {
	notFound Get  `path:"/notfound"`
	get      Get  `path:"/get/:id"`
	send     Post `path:"/send/:id"`
}

func (d *Succeed) NotFound(ctx Context) (string, error) { return "", NotFound("not found") }
func (d *Succeed) Get(ctx Context) (string, error) {
	ctx.ResponseHeader().Set("Custom", "value")
	return "get" + ctx.Query().Get("id") + ctx.Query().Get("extra"), nil
}
func (d *Succeed) Send(ctx Context, i string) (string, error) {
	ctx.Broadcast("123", "room"+ctx.Query().Get("id")+":"+i)
	return "", nil
}

func TestStreamingGet(t *testing.T) {
	ht := &HTTPTransport{
		DefaultMime:           "application/json",
		DefaultCharset:        "utf-8",
		StreamingNotification: true,
		MaxConnectionPerRoom:  2,
	}
	handler, err := New(new(Succeed))
	assert.MustEqual(t, err, nil)
	err = ht.Add("/prefix", handler)
	assert.MustEqual(t, err, nil)
	req, err := http.NewRequest("WATCH", "http://domain", nil)
	assert.MustEqual(t, err, nil)

	type Test struct {
		url   string
		type_ string
		ret   interface{}
		ok    bool
	}
	var tests = []Test{
		{"/prefix/get/123", "/prefix/get", "get123", true},
		{"/prefix/get/123?extra=abc", "/prefix/get", "get123abc", true},

		{"/prefix/notfound", "", "", false},
		{"/nonexist", "", "", false},
	}
	for i, test := range tests {
		ret, err := ht.streamingGet(test.url, req)
		assert.MustEqual(t, err == nil, test.ok, "test %d", i)
		if err != nil {
			continue
		}
		assert.Equal(t, ret, test.ret, "test %d", i)
	}
}

func TestStreaming(t *testing.T) {
	handler, err := New(new(Succeed))
	assert.MustEqual(t, err, nil)
	assert.Equal(t, len(handler.routes), 3)

	ht := &HTTPTransport{
		DefaultMime:           "application/json",
		DefaultCharset:        "utf-8",
		StreamingNotification: true,
		MaxConnectionPerRoom:  1,
		StreamingPingInterval: time.Second,
	}
	err = ht.Add("/prefix", handler)
	assert.MustEqual(t, err, nil)

	server := httptest.NewServer(ht)
	defer server.Close()

	req, err := http.NewRequest("WATCH", server.URL+"?room=123&room=abc", nil)
	assert.MustEqual(t, err, nil)
	streaming, err := http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.MustEqual(t, streaming.StatusCode, http.StatusOK)
	assert.MustContain(t, streaming.Header, "Connection")
	assert.MustEqual(t, streaming.Header["Connection"], []string{"keep-alive"})
	assert.MustContain(t, streaming.Header, "Content-Type")
	assert.MustEqual(t, streaming.Header["Content-Type"], []string{"application/json"})

	time.Sleep(ht.StreamingPingInterval * 3 / 2)

	resp, err := http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.MustEqual(t, resp.StatusCode, http.StatusBadRequest)
	resp.Body.Close()

	streaming.Body.Close()
	time.Sleep(ht.StreamingPingInterval)

	resp, err = http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.MustEqual(t, streaming.StatusCode, http.StatusOK)
	resp.Body.Close()
}

func TestSucceed(t *testing.T) {
	handler, err := New(new(Succeed))
	assert.MustEqual(t, err, nil)
	assert.Equal(t, len(handler.routes), 3)

	ht := &HTTPTransport{
		DefaultMime:           "application/json",
		DefaultCharset:        "utf-8",
		StreamingNotification: true,
		MaxConnectionPerRoom:  2,
		StreamingPingInterval: time.Second,
	}
	err = ht.Add("/prefix", handler)
	assert.MustEqual(t, err, nil)

	server := httptest.NewServer(ht)
	defer server.Close()

	req, err := http.NewRequest("NOMETHOD", server.URL+"/prefix/get/123", nil)
	assert.MustEqual(t, err, nil)
	resp, err := http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusMethodNotAllowed)
	assert.MustContain(t, resp.Header, "Allow")
	assert.MustContain(t, resp.Header["Allow"], "GET")
	resp.Body.Close()

	req, err = http.NewRequest("NOMETHOD", server.URL+"/prefix/get/123?_method=GET", nil)
	assert.MustEqual(t, err, nil)
	resp, err = http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()

	req, err = http.NewRequest("WATCH", server.URL+"?room=123&room=abc&init=/prefix/get/123&init=/prefix/send/123&init=/prefix/notfound", nil)
	assert.MustEqual(t, err, nil)
	streaming, err := http.DefaultClient.Do(req)
	assert.MustEqual(t, err, nil)
	assert.MustEqual(t, streaming.StatusCode, http.StatusOK)
	assert.MustContain(t, streaming.Header, "Connection")
	assert.MustEqual(t, streaming.Header["Connection"], []string{"keep-alive"})
	assert.MustContain(t, streaming.Header, "Content-Type")
	assert.MustEqual(t, streaming.Header["Content-Type"], []string{"application/json"})

	defer streaming.Body.Close()
	buf := make([]byte, 1024)
	n, err := streaming.Body.Read(buf)
	assert.MustEqual(t, err, nil)
	buf = buf[:n]
	assert.Equal(t, string(buf), "\"get123\"\n")

	resp, err = http.Get(server.URL + "/prefix/get/123")
	assert.MustEqual(t, err, nil)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	decoder := json.NewDecoder(resp.Body)
	var ret string
	err = decoder.Decode(&ret)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, ret, "get123")
	assert.MustContain(t, resp.Header, "Custom")
	assert.Equal(t, resp.Header["Custom"], []string{"value"})

	resp, err = http.Get(server.URL + "/prefix/notfound")
	assert.MustEqual(t, err, nil)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)
	b, err := ioutil.ReadAll(resp.Body)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, string(b), "not found\n")

	resp, err = http.Get(server.URL + "/prefix/notexist")
	assert.MustEqual(t, err, nil)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)
	b, err = ioutil.ReadAll(resp.Body)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, string(b), "\n")

	post := bytes.NewBufferString(`"123"`)
	resp, err = http.Post(server.URL+"/prefix/send/xyz", "application/json", post)
	assert.MustEqual(t, err, nil)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusAccepted)
	decoder = json.NewDecoder(resp.Body)
	err = decoder.Decode(&ret)
	assert.MustEqual(t, err, nil)
	assert.Equal(t, ret, "")

	buf = make([]byte, 1024)
	n, err = streaming.Body.Read(buf)
	assert.MustEqual(t, err, nil)
	buf = buf[:n]
	assert.MustEqual(t, string(buf), "\"roomxyz:123\"\n")
}
