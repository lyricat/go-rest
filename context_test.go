package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestBaseContextNew(t *testing.T) {
	type Test struct {
		handlerName string
		marshaller  Marshaller
		charset     string
		vars        map[string]string

		targetMarshaller Marshaller
	}
	fakeMarshaller := FakeMarshaller{}
	RegisterMarshaller("fake/marshaller", fakeMarshaller)
	var tests = []Test{
		{"handler1", nil, "utf-8", nil, jsonMarshaller},
		{"handler1", fakeMarshaller, "utf-8", nil, fakeMarshaller},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "http://domain/path", nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		resp := httptest.NewRecorder()
		ctx := newBaseContext(test.handlerName, test.marshaller, test.charset, test.vars, req, resp)
		assert.Equal(t, ctx.handlerName, test.handlerName, "test %d", i)
		assert.Equal(t, ctx.marshaller, test.targetMarshaller, "test %d", i)
	}
}

func TestBaseContextIfMatch(t *testing.T) {
	type Test struct {
		header http.Header
		etag   string
		ok     bool
	}
	var tests = []Test{
		{http.Header{"If-Match": []string{`"737060cd8c284d8af7ad3082f209582d"`}}, "737060cd8c284d8af7ad3082f209582d", true},
		{http.Header{"If-Match": []string{`"737060cd8c284d8af7ad3082f209582e"`}}, "737060cd8c284d8af7ad3082f209582d", false},
		{http.Header{"If-Match": []string{`*`}}, "737060cd8c284d8af7ad3082f209582d", true},
		{http.Header{"If-Match": []string{`"737060cd8c284d8af7ad3082f209582d", "737060cd8c284d8af7ad3082f209582e"`}}, "737060cd8c284d8af7ad3082f209582d", true},
		{http.Header{"If-Match": []string{`"737060cd8c284d8af7ad3082f209582c", "737060cd8c284d8af7ad3082f209582d"`}}, "737060cd8c284d8af7ad3082f209582d", true},
		{http.Header{"If-Match": []string{`"737060cd8c284d8af7ad3082f209582c", "737060cd8c284d8af7ad3082f209582e"`}}, "737060cd8c284d8af7ad3082f209582d", false},
		{nil, "737060cd8c284d8af7ad3082f209582d", false},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "http://domain/path", nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		req.Header = test.header
		resp := httptest.NewRecorder()
		ctx := newBaseContext("test", nil, "", nil, req, resp)
		assert.Equal(t, ctx.IfMatch(test.etag), test.ok, "test %d", i)
	}
}

func TestBaseContextIfNoneMatch(t *testing.T) {
	type Test struct {
		header http.Header
		etag   string
		ok     bool
	}
	var tests = []Test{
		{http.Header{"If-None-Match": []string{`"737060cd8c284d8af7ad3082f209582d"`}}, "737060cd8c284d8af7ad3082f209582d", false},
		{http.Header{"If-None-Match": []string{`"737060cd8c284d8af7ad3082f209582e"`}}, "737060cd8c284d8af7ad3082f209582d", true},
		{http.Header{"If-None-Match": []string{`*`}}, "737060cd8c284d8af7ad3082f209582d", false},
		{http.Header{"If-None-Match": []string{`"737060cd8c284d8af7ad3082f209582d", "737060cd8c284d8af7ad3082f209582e"`}}, "737060cd8c284d8af7ad3082f209582d", false},
		{http.Header{"If-None-Match": []string{`"737060cd8c284d8af7ad3082f209582c", "737060cd8c284d8af7ad3082f209582d"`}}, "737060cd8c284d8af7ad3082f209582d", false},
		{http.Header{"If-None-Match": []string{`"737060cd8c284d8af7ad3082f209582c", "737060cd8c284d8af7ad3082f209582e"`}}, "737060cd8c284d8af7ad3082f209582d", true},
		{nil, "737060cd8c284d8af7ad3082f209582d", true},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "http://domain/path", nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		req.Header = test.header
		resp := httptest.NewRecorder()
		ctx := newBaseContext("test", nil, "", nil, req, resp)
		assert.Equal(t, ctx.IfNoneMatch(test.etag), test.ok, "test %d", i)
	}
}

func TestBaseContextReturn(t *testing.T) {
	type Test struct {
		code       int
		fmtAndArgs []interface{}

		body string
	}
	var tests = []Test{
		{http.StatusOK, nil, ""},
		{http.StatusBadRequest, []interface{}{"test"}, "test\n"},
		{http.StatusBadRequest, []interface{}{"test %s", "bad"}, "test bad\n"},
		{http.StatusBadRequest, []interface{}{fmt.Errorf("some error")}, "some error\n"},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "http://domain/path", nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		resp := httptest.NewRecorder()
		ctx := newBaseContext("test", nil, "", nil, req, resp)
		ctx.Return(test.code, test.fmtAndArgs...)
		assert.Equal(t, resp.Code, test.code, "test %d", i)
		assert.Equal(t, resp.Body.String(), test.body, "test %d", i)
	}
}

func TestBaseContextRender(t *testing.T) {
	fakeMarshaller := FakeMarshaller{}
	RegisterMarshaller("fake/marshaller", fakeMarshaller)
	type Test struct {
		code int
		v    interface{}

		body string
	}
	var tests = []Test{
		{http.StatusOK, 1, "fake marshal writed: 1"},
		{http.StatusOK, "str", "fake marshal writed: \"str\""},
		{http.StatusOK, []int{1, 2, 3}, "fake marshal writed: []int{1, 2, 3}"},
		{http.StatusOK, map[string]int{"a": 1}, "fake marshal writed: map[string]int{\"a\":1}"},
		{http.StatusOK, fakeMarshaller, "fake marshal writed: rest.FakeMarshaller{}"},
		{http.StatusBadRequest, fmt.Errorf("invalid input"), "fake marshal writed: &errors.errorString{s:\"invalid input\"}"},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "http://domain/path", nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		resp := httptest.NewRecorder()
		ctx := newBaseContext("test", fakeMarshaller, "", nil, req, resp)
		ctx.Return(test.code)
		ctx.Render(test.v)
		assert.Equal(t, resp.Code, test.code, "test %d", i)
		assert.Equal(t, resp.Body.String(), test.body, "test %d", i)
	}
}

func TestBaseContextBind(t *testing.T) {
	type Test struct {
		vars  map[string]string
		url   string
		id    string
		v     interface{}
		ok    bool
		value string
	}
	var bl bool
	var str string
	var i int
	var i64 int64
	var i32 int32
	var i16 int16
	var i8 int8
	var b byte
	var u uint
	var u64 uint64
	var u32 uint32
	var u16 uint16
	var u8 uint8
	var f32 float32
	var f64 float64
	var astr []string
	var ai []int
	var ai64 []int64
	var ai32 []int32
	var ai16 []int16
	var ai8 []int8
	var ab []byte
	var au []uint
	var au64 []uint64
	var au32 []uint32
	var au16 []uint16
	var au8 []uint8
	var af32 []float32
	var af64 []float64
	var ft Test
	var tests = []Test{
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &ft, false, ""},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &bl, true, "true"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "nonexist", &bl, true, "false"},
		{map[string]string{"str": "url_string"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &bl, true, "true"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &str, true, "some_string"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i, false, "some_string"},
		{map[string]string{"str": "url_string"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &str, true, "url_string"},
		{map[string]string{"str": "url_string"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i, false, "url_string"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &str, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i, true, "1"},
		{map[string]string{"i": "2"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i, true, "2"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i64, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i32, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i16, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i8, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &b, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u64, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u32, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u16, true, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u8, true, "1"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &f32, true, "1"},
		{map[string]string{"i": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &f32, true, "2.1"},
		{map[string]string{"i": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i, false, "2.1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &f64, true, "1"},
		{map[string]string{"i": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &f64, true, "2.1"},
		{map[string]string{"i": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i, false, "2.1"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &astr, true, "[a b]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai, false, ""},
		{map[string]string{"sarray": "z"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &astr, true, "[z a b]"},
		{map[string]string{"sarray": "z"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &ai, false, ""},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &astr, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai, true, "[1 2]"},
		{map[string]string{"iarray": "0"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai, true, "[0 1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai64, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai32, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai16, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai8, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ab, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au64, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au32, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au16, true, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au8, true, "[1 2]"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af32, true, "[1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af32, true, "[2.1 1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai, false, ""},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af64, true, "[1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af64, true, "[2.1 1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai, false, ""},

		// failed convert
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i, false, "1"},
		{map[string]string{"i": "2"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i, false, "2"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i64, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i32, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i16, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &i8, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &b, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &u, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &u64, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &u32, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &u16, false, "1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &u8, false, "1"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &f32, false, "1"},
		{map[string]string{"i": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &f32, false, "2.1"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &f64, false, "1"},
		{map[string]string{"i": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &f64, false, "2.1"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai, false, "[1 2]"},
		{map[string]string{"iarray": "0"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai, false, "[0 1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai64, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai32, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai16, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ai8, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &ab, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &au, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &au64, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &au32, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &au16, false, "[1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &au8, false, "[1 2]"},

		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &af32, false, "[1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &af32, false, "[2.1 1 2]"},
		{nil, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &af64, false, "[1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &af64, false, "[2.1 1 2]"},

		// failed query
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &bl, false, "true"},

		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "str", &str, false, "some_string"},

		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &str, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i64, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i32, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i16, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &i8, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &b, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u64, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u32, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u16, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &u8, false, "1"},

		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &f32, false, "1"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "i", &f64, false, "1"},

		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &astr, false, "[a b]"},
		{map[string]string{"sarray": "z"}, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "sarray", &astr, false, "[z a b]"},

		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &astr, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai, false, "[1 2]"},
		{map[string]string{"iarray": "0"}, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai, false, "[0 1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai64, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai32, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai16, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ai8, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &ab, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au64, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au32, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au16, false, "[1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &au8, false, "[1 2]"},

		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af32, false, "[1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af32, false, "[2.1 1 2]"},
		{nil, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af64, false, "[1 2]"},
		{map[string]string{"iarray": "2.1"}, "http://domain/path?str=%some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", "iarray", &af64, false, "[2.1 1 2]"},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", test.url, nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		resp := httptest.NewRecorder()
		var ctx Context
		ctx = newBaseContext("testHandler", nil, "application/json", test.vars, req, resp)
		assert.Equal(t, ctx.Request(), req, "test %d", i)
		assert.Equal(t, ctx.Response(), resp, "test %d", i)
		ctx.Bind(test.id, test.v)
		assert.Equal(t, ctx.BindError() == nil, test.ok, "test %d, err: %s", i, ctx.BindError())
		if ctx.BindError() != nil {
			continue
		}
		v := reflect.ValueOf(test.v)
		assert.Equal(t, fmt.Sprintf("%v", v.Elem().Interface()), test.value, "test %d", i)
	}
}

func TestBaseContextBindError(t *testing.T) {
	req, err := http.NewRequest("GET", "http://domain/path?str=some_string&i=1&iarray=1&iarray=2&sarray=a&sarray=b", nil)
	assert.MustEqual(t, err, nil)
	resp := httptest.NewRecorder()
	var ctx Context
	ctx = newBaseContext("testHandler", nil, "application/json", nil, req, resp)
	var i int
	var s string
	ctx.Bind("str", &s)
	assert.Equal(t, ctx.BindError(), nil)
	ctx.Bind("str", &i)
	assert.NotEqual(t, ctx.BindError(), nil)
	err = ctx.BindError()
	ctx.Bind("str", &s)
	assert.Equal(t, ctx.BindError(), err)
	ctx.BindReset()
	ctx.Bind("str", &s)
	assert.Equal(t, ctx.BindError(), nil)
}
