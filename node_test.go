package rest

import (
	"bytes"
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMapFormatter(t *testing.T) {
	type Test struct {
		prefix    string
		path      string
		args      map[string]string
		formatter string
		url       string
	}
	var tests = []Test{
		{"/", "path", nil, "/path", "/path"},
		{"", "path", nil, "/path", "/path"},
		{"", "", nil, "/", "/"},
		{"/prefix", "", nil, "/prefix", "/prefix"},
		{"prefix", "path", nil, "/prefix/path", "/prefix/path"},
		{"/prefix", "/path", nil, "/prefix/path", "/prefix/path"},
		{"/prefix/", "/path", nil, "/prefix/path", "/prefix/path"},
		{"", "/:id", map[string]string{"id": "123"}, "/:id", "/123"},
		{"", "/:id/:key", map[string]string{"id": "123", "key": "abc"}, "/:id/:key", "/123/abc"},
	}
	for i, test := range tests {
		formatter := pathToFormatter(test.prefix, test.path)
		assert.Equal(t, string(formatter), test.formatter, fmt.Sprintf("test %d", i))
		assert.Equal(t, formatter.pathMap(test.args), test.url, fmt.Sprintf("test %d", i))
	}
}

func TestFormatter(t *testing.T) {
	type Test struct {
		prefix    string
		path      string
		args      []string
		formatter string
		url       string
	}
	var tests = []Test{
		{"/", "path", nil, "/path", "/path"},
		{"", "path", nil, "/path", "/path"},
		{"prefix", "path", nil, "/prefix/path", "/prefix/path"},
		{"/prefix", "/path", nil, "/prefix/path", "/prefix/path"},
		{"", "/:id", []string{"id", "123"}, "/:id", "/123"},
		{"", "/:id/:key", []string{"id", "123", "key", "abc"}, "/:id/:key", "/123/abc"},
	}
	for i, test := range tests {
		formatter := pathToFormatter(test.prefix, test.path)
		assert.Equal(t, string(formatter), test.formatter, fmt.Sprintf("test %d", i))
		assert.Equal(t, formatter.path(test.args...), test.url, fmt.Sprintf("test %d", i))
	}
}

func TestProcessorNodeHandle(t *testing.T) {
	type Test struct {
		findex       int
		requestType  reflect.Type
		responseType reflect.Type
		requestBody  string

		code         int
		fname        string
		input        string
		responseBody string
	}
	s := new(FakeProcessor)
	s.last = make(map[string]string)
	instance := reflect.ValueOf(s).Elem()
	instanceType := instance.Type()
	nino, ok := instanceType.MethodByName("NoInputNoOutput")
	if !ok {
		t.Fatal("no NoInputNoOutput")
	}
	ni, ok := instanceType.MethodByName("NoInput")
	if !ok {
		t.Fatal("no NoInput")
	}
	no, ok := instanceType.MethodByName("NoOutput")
	if !ok {
		t.Fatal("no NoOutput")
	}
	n, ok := instanceType.MethodByName("Normal")
	if !ok {
		t.Fatal("no Normal")
	}

	var tests = []Test{
		{nino.Index, nil, nil, "", http.StatusOK, "NoInputNoOutput", "", ""},
		{ni.Index, nil, reflect.TypeOf(""), "", http.StatusOK, "NoInput", "", "\"output\"\n"},
		{no.Index, reflect.TypeOf(""), reflect.TypeOf(""), "\"input\"", http.StatusOK, "NoOutput", "input", ""},
		{n.Index, reflect.TypeOf(""), reflect.TypeOf(""), "\"input\"", http.StatusOK, "Normal", "input", "\"output\"\n"},
	}
	for i, test := range tests {
		node := processorNode{
			funcIndex:    test.findex,
			requestType:  test.requestType,
			responseType: test.responseType,
		}
		buf := bytes.NewBufferString(test.requestBody)
		req, err := http.NewRequest("GET", "http://fake.domain", buf)
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		w := httptest.NewRecorder()
		w.Code = http.StatusOK
		ctx, err := newContext(w, req, nil, "application/json", "utf-8")
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		node.handle(instance, ctx)
		assert.Equal(t, w.Code, test.code, fmt.Sprintf("test %d code: %d", i, w.Code))
		assert.Equal(t, w.Body.String(), test.responseBody, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["method"], test.fname, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["input"], test.input, fmt.Sprintf("test %d", i))
	}
}

func TestStreamingNodeHandle(t *testing.T) {
	type Test struct {
		f           reflect.Method
		end         string
		requestType reflect.Type
		requestBody string

		code   int
		method string
		input  string
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
		{ni, "", nil, "", http.StatusOK, "NoInput", ""},
		{i, "\n", reflect.TypeOf(""), "\"input\"", http.StatusOK, "Input", "input"},
	}
	for i, test := range tests {
		sn := &streamingNode{
			funcIndex:   test.f.Index,
			end:         test.end,
			requestType: test.requestType,
		}
		buf := bytes.NewBufferString(test.requestBody)
		req, err := http.NewRequest("GET", "http://fake.domain", buf)
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		h := newHijacker()
		ctx, err := newContext(h, req, nil, "application/json", "utf-8")
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		sn.handle(instance, ctx)
		assert.Equal(t, h.code, test.code, fmt.Sprintf("test %d code: %d", i, h.code))
		assert.Equal(t, s.last["method"], test.f.Name, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["input"], test.input, fmt.Sprintf("test %d", i))
	}
}
