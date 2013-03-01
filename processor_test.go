package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"reflect"
	"regexp"
	"testing"
)

type TestProcessor struct{}

type MyInt int

func (t TestProcessor) HandlerNoPost(a string, b int, c MyInt) string {
	return ""
}

type Post struct {
	a int
}

func (t TestProcessor) HandlerWithPost(a string, b int, c Post) string {
	return ""
}

func (t TestProcessor) HandlerInvalid(b bool) {}

func (t TestProcessor) HandlerInvalid2() (string, int) { return "", 0 }

func (t TestProcessor) InvalidMarsharl(f func()) func() { return f }

func mustGet(method reflect.Method, ok bool) reflect.Method {
	if !ok {
		panic("can't find method")
	}
	return method
}

func TestProcessorInit(t *testing.T) {
	processor := reflect.TypeOf(TestProcessor{})
	type Test struct {
		root string
		tag  reflect.StructTag
		f    reflect.Method

		ok           bool
		method       string
		path         string
		kinds        string
		requestType  string
		responseType string
	}

	var tests = []Test{
		{"/root", `method:"GET" path:"/hello/(.*?)/(.*?)/(.*?)"`, mustGet(processor.MethodByName("HandlerNoPost")), true, "GET", "/root/hello/(.*?)/(.*?)/(.*?)$", "[string int int]", "<nil>", "string"},
		{"/", `method:"POST" path:"/hello/(.*?)/(.*?)"`, mustGet(processor.MethodByName("HandlerWithPost")), true, "POST", "/hello/(.*?)/(.*?)$", "[string int]", "rest.Post", "string"},
		{"/", `method:"POST" path:"/hello/(.*?)"`, mustGet(processor.MethodByName("HandlerInvalid")), false, "", "", "", "", ""},
	}

	for i, test := range tests {
		processor := new(Processor)
		err := initProcessor(test.root, reflect.ValueOf(processor).Elem(), test.tag, test.f)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			continue
		} else if err != nil {
			t.Error(err)
		}
		assert.Equal(t, processor.method, test.method, fmt.Sprintf("test %d", i))
		assert.Equal(t, processor.Method, test.method, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%s", processor.path), test.path, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%s", processor.pathArgKinds), test.kinds, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", processor.requestType), test.requestType, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", processor.responseType), test.responseType, fmt.Sprintf("test %d", i))
		assert.Equal(t, processor.Tag, test.tag, fmt.Sprintf("test %d", i))
		assert.Equal(t, processor.funcIndex, test.f.Index, fmt.Sprintf("test %d", i))
	}
}

func TestParsePath(t *testing.T) {
	type Test struct {
		root   string
		path   string
		ok     bool
		expect string
	}
	var tests = []Test{
		{"/root", "/hello/(.*?)", true, "/root/hello/(.*?)$"},
		{"/", "(.*?)", true, "/(.*?)$"},
	}

	for i, test := range tests {
		regexp, err := parsePath(test.root, test.path)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			continue
		}
		assert.Equal(t, regexp.String(), test.expect, fmt.Sprintf("test %d", i))
	}
}

func TestParseResponse(t *testing.T) {
	processor := reflect.TypeOf(TestProcessor{})
	type Test struct {
		f    reflect.Type
		ok   bool
		resp reflect.Type
	}
	var tests = []Test{
		{mustGet(processor.MethodByName("HandlerNoPost")).Type, true, reflect.TypeOf("")},
		{mustGet(processor.MethodByName("HandlerWithPost")).Type, true, reflect.TypeOf("")},
		{mustGet(processor.MethodByName("HandlerInvalid")).Type, true, reflect.TypeOf(nil)},
		{mustGet(processor.MethodByName("HandlerInvalid2")).Type, false, reflect.TypeOf(nil)},
		{mustGet(processor.MethodByName("InvalidMarsharl")).Type, false, reflect.TypeOf(nil)},
	}

	for i, test := range tests {
		resp, err := parseResponseType(test.f)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			continue
		} else if err != nil {
			t.Error(err)
		}
		if test.resp != nil {
			assert.Equal(t, resp.String(), test.resp.String(), fmt.Sprintf("test %d", i))
		} else {
			assert.Equal(t, resp, nil, fmt.Sprintf("test %d", i))
		}
	}
}

func TestParseRequest(t *testing.T) {
	processor := reflect.TypeOf(TestProcessor{})
	type Test struct {
		path  string
		f     reflect.Type
		ok    bool
		kinds string
		post  reflect.Type
	}
	var tests = []Test{
		{"/hello/(.*?)/(.*?)/(.*?)", mustGet(processor.MethodByName("HandlerNoPost")).Type, true, "[string int int]", reflect.TypeOf(nil)},
		{"/hello/(.*?)/(.*?)", mustGet(processor.MethodByName("HandlerWithPost")).Type, true, "[string int]", reflect.TypeOf(Post{})},
		{"/hello/(.*?)", mustGet(processor.MethodByName("HandlerInvalid")).Type, false, "", reflect.TypeOf(nil)},
		{"/hello", mustGet(processor.MethodByName("InvalidMarsharl")).Type, false, "", reflect.TypeOf(nil)},
	}

	for i, test := range tests {
		kinds, post, err := parseRequestType(regexp.MustCompile(test.path), test.f)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			continue
		}
		if test.post != nil {
			assert.Equal(t, post.String(), test.post.String(), fmt.Sprintf("test %d", i))
		} else {
			assert.Equal(t, post, nil, fmt.Sprintf("test %d", i))
		}
		assert.Equal(t, fmt.Sprintf("%v", kinds), test.kinds, fmt.Sprintf("test %d", i))
	}
}

func TestProcessorGetArg(t *testing.T) {
	processor := reflect.ValueOf(TestProcessor{})
	type Test struct {
		root   string
		tag    reflect.StructTag
		method reflect.Method
		f      reflect.Value
		path   string
		ok     bool
		args   string
	}

	var tests = []Test{
		{"/root", `method:"GET" path:"/hello/(.*?)/(.*?)/(.*?)"`, mustGet(processor.Type().MethodByName("HandlerNoPost")), processor.MethodByName("HandlerNoPost"), "/root/hello/abc/123/456", true, "[abc 123 456]"},
		{"/", `method:"POST" path:"/hello/(.*?)/(.*?)"`, mustGet(processor.Type().MethodByName("HandlerWithPost")), processor.MethodByName("HandlerWithPost"), "/hello/abc/123", true, "[abc 123]"},
	}

	for i, test := range tests {
		processor := new(Processor)
		err := initProcessor(test.root, reflect.ValueOf(processor).Elem(), test.tag, test.method)
		if err != nil {
			panic(err)
		}
		args, err := processor.getArgs(test.path)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if err != nil {
			continue
		}
		assert.Equal(t, valuesToString(args), test.args, fmt.Sprintf("test %d", i))
	}
}

func TestProcessorPath(t *testing.T) {
	processor := reflect.ValueOf(TestProcessor{})
	type Test struct {
		root   string
		tag    reflect.StructTag
		method reflect.Method
		f      reflect.Value
		args   []interface{}
		ok     bool
		path   string
	}

	var tests = []Test{
		{"/root", `method:"GET" path:"/hello/(.*?)/([0-9]+)/(.*?)"`, mustGet(processor.Type().MethodByName("HandlerNoPost")), processor.MethodByName("HandlerNoPost"), []interface{}{"abc", 123, 456}, true, "/root/hello/abc/123/456"},
		{"/", `method:"POST" path:"/hello/(.*?)/(.*?)"`, mustGet(processor.Type().MethodByName("HandlerWithPost")), processor.MethodByName("HandlerWithPost"), []interface{}{"abc", 123}, true, "/hello/abc/123"},
		{"/", `method:"POST" path:"/hello/(.*?)/(.*?)"`, mustGet(processor.Type().MethodByName("HandlerWithPost")), processor.MethodByName("HandlerWithPost"), []interface{}{"abc"}, false, ""},
	}

	for i, test := range tests {
		processor := new(Processor)
		err := initProcessor(test.root, reflect.ValueOf(processor).Elem(), test.tag, test.method)
		if err != nil {
			panic(err)
		}
		path, err := processor.Path(test.args...)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			continue
		} else if err != nil {
			t.Error(err)
		}
		assert.Equal(t, path, test.path, fmt.Sprintf("test %d", i))
	}
}

func valuesToString(args []reflect.Value) string {
	ret := "["
	for _, a := range args {
		ret += fmt.Sprintf("%v ", a.Interface())
	}
	if ret[len(ret)-1] == ' ' {
		ret = ret[:len(ret)-1]
	}
	ret += "]"
	return ret
}
