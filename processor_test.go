package rest

import (
	"bytes"
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"net/http"
	"reflect"
	"testing"
)

type FakeProcessor struct {
	last map[string]string
}

func (f FakeProcessor) NoInputNoOutput() {
	f.last["method"] = "NoInputNoOutput"
	f.last["input"] = ""
	f.last["output"] = ""
}

func (f FakeProcessor) NoInput() string {
	f.last["method"] = "NoInput"
	f.last["input"] = ""
	f.last["output"] = "output"
	return "output"
}

func (f FakeProcessor) NoOutput(post string) {
	f.last["method"] = "NoOutput"
	f.last["input"] = post
	f.last["output"] = ""
}

func (f FakeProcessor) Normal(post string) string {
	f.last["method"] = "Normal"
	f.last["input"] = post
	f.last["output"] = "output"
	return "output"
}

func (f FakeProcessor) ErrorInput(a, b int) {}

func (f FakeProcessor) ErrorOutput() (string, string) {
	return "", ""
}

func TestNewProcessor(t *testing.T) {
	type Test struct {
		path pathFormatter
		f    reflect.Method
		tag  reflect.StructTag

		ok        bool
		funcIndex int
		request   string
		response  string
	}
	s := new(FakeProcessor)
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
	ei, ok := instanceType.MethodByName("ErrorInput")
	if !ok {
		t.Fatal("no ErrorInput")
	}
	eo, ok := instanceType.MethodByName("ErrorOutput")
	if !ok {
		t.Fatal("no ErrorOutput")
	}
	var tests = []Test{
		{"/", nino, "", true, nino.Index, "<nil>", "<nil>"},
		{"/", ni, "", true, ni.Index, "<nil>", "string"},
		{"/", no, "", true, no.Index, "string", "<nil>"},
		{"/", n, "", true, n.Index, "string", "string"},
		{"/", ei, "", false, ei.Index, "", ""},
		{"/", eo, "", false, eo.Index, "", ""},
	}
	for i, test := range tests {
		inner := new(innerProcessor)
		err := inner.init(test.path, test.f, test.tag)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			continue
		}
		assert.Equal(t, inner.formatter, test.path, fmt.Sprintf("test %d", i))
		assert.Equal(t, inner.funcIndex, test.funcIndex, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", inner.requestType), test.request, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", inner.responseType), test.response, fmt.Sprintf("test %d", i))
	}
}

func TestProcessorHandle(t *testing.T) {
	type Test struct {
		path        string
		f           reflect.Method
		tag         reflect.StructTag
		requestBody string

		code         int
		vars         map[string]string
		input        string
		output       string
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
		{"/", nino, "", "", http.StatusOK, nil, "", "", ""},
		{"/", ni, "", "", http.StatusOK, nil, "", "output", "\"output\"\n"},
		{"/", no, "", "\"input\"", http.StatusOK, nil, "input", "", ""},
		{"/", n, "", "\"input\"", http.StatusOK, nil, "input", "output", "\"output\"\n"},
	}
	for i, test := range tests {
		inner := new(innerProcessor)
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
		w := newWriter()
		ctx, err := newContext(w, req, test.vars, "application/json", "utf-8")
		assert.Equal(t, err, nil, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil {
			continue
		}
		inner.handle(instance, ctx)
		assert.Equal(t, w.code, http.StatusOK, fmt.Sprintf("test %d code: %d", i, w.code))
		assert.Equal(t, w.buf.String(), test.responseBody, fmt.Sprintf("test %d", i))
		assert.Equal(t, inner.formatter, test.path, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["method"], test.f.Name, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["input"], test.input, fmt.Sprintf("test %d", i))
		assert.Equal(t, s.last["output"], test.output, fmt.Sprintf("test %d", i))
	}
}

type fakeResponseWriter struct {
	code   int
	header http.Header
	buf    *bytes.Buffer
}

func newWriter() *fakeResponseWriter {
	return &fakeResponseWriter{
		code:   http.StatusOK,
		header: make(http.Header),
		buf:    bytes.NewBuffer(nil),
	}
}

func (w *fakeResponseWriter) Header() http.Header {
	return w.header
}

func (w *fakeResponseWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *fakeResponseWriter) WriteHeader(code int) {
	w.code = code
}
