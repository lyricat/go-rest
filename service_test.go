package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"net/http"
	"reflect"
	"testing"
)

func TestService(t *testing.T) {
	type Test struct {
		tag reflect.StructTag

		ok      bool
		prefix  string
		realm   string
		mime    string
		charset string
	}
	var tests = []Test{
		{``, true, "/", "[]", "application/json", "utf-8"},
		{`prefix:"/prefix" realm:"abc,xyz" mime:"application/xml" charset:"gbk"`, true, "/prefix", "[abc xyz]", "application/xml", "gbk"},
		{`prefix:"/prefix" realm:"abc,xyz" charset:"gbk"`, true, "/prefix", "[abc xyz]", "application/json", "gbk"},
		{`prefix:"/prefix" realm:"abc,xyz" mime:"application/xml"`, true, "/prefix", "[abc xyz]", "application/xml", "utf-8"},
		{`realm:"abc,xyz" mime:"application/xml"`, true, "/", "[abc xyz]", "application/xml", "utf-8"},
	}

	for i, test := range tests {
		service := new(Service)
		err := initService(reflect.ValueOf(service).Elem(), test.tag)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			t.Error(err)
			continue
		}
		assert.Equal(t, service.prefix, test.prefix, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.Prefix, test.prefix, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.defaultMime, test.mime, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultCharset, test.charset, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.defaultMime, test.mime, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultCharset, test.charset, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.Tag, test.tag, fmt.Sprintf("test %d", i))
	}

	for i, test := range tests {
		service := new(Service)
		service.Prefix = "/abcde"
		service.DefaultMime = "text/plain"
		service.DefaultCharset = "abc"
		err := initService(reflect.ValueOf(service).Elem(), test.tag)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			t.Error(err)
			continue
		}
		assert.Equal(t, service.prefix, "/abcde", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.Prefix, "/abcde", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.defaultMime, "text/plain", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultCharset, "abc", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.defaultMime, "text/plain", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultCharset, "abc", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.Tag, test.tag, fmt.Sprintf("test %d", i))
	}
}

func TestGetContentType(t *testing.T) {
	type Test struct {
		contentType string
		mime        string
		charset     string
	}
	var tests = []Test{
		{"", "application/json", "utf-8"},
		{"application/xml", "application/xml", "utf-8"},
		{"application/xml; charset=gbk", "application/xml", "gbk"},
		{"application/xml; other=abc; charset=gbk", "application/xml", "gbk"},
		{"application/xml; other=abc", "application/xml", "utf-8"},
	}

	service := new(Service)
	err := initService(reflect.ValueOf(service).Elem(), `mime:"application/json" charset:"utf-8"`)
	if err != nil {
		t.Fatal(err)
	}
	for i, test := range tests {
		req, _ := http.NewRequest("GET", "http://127.0.0.1/", nil)
		req.Header.Set("Content-Type", test.contentType)
		mime, charset := service.innerService.getContentTypeFromRequset(req)
		assert.Equal(t, mime, test.mime, fmt.Sprintf("test %d", i))
		assert.Equal(t, charset, test.charset, fmt.Sprintf("test %d", i))
	}
}

func TestParseRealm(t *testing.T) {
	type Test struct {
		realm  reflect.StructTag
		result string
	}
	var tests = []Test{
		{`realm:"abc,123"`, "[abc 123]"},
	}

	for i, test := range tests {
		realm := parseRealm(test.realm)
		assert.Equal(t, fmt.Sprintf("%v", realm), test.result, fmt.Sprintf("test %d", i))
	}
}

type FindProcessor struct{}

func (t FindProcessor) handler1()                {}
func (t FindProcessor) handler2(a string)        {}
func (t FindProcessor) handler3(b string, c int) {}

func TestFindHandler(t *testing.T) {
	processor := reflect.TypeOf(FindProcessor{})
	type Test struct {
		method string
		url    string
		ok     bool
		result string
	}
	var tests = []Test{
		{"GET", "/path", true, "/path$"},
		{"GET", "/path/abc", true, "/path/([^/]*?)$"},
		{"GET", "/path/abc/123", true, "/path/([^/]*?)/([0-9]*)$"},
		{"GET", "/path1", false, ""},
		{"GET", "/path/abc/xyz", false, ""},
		{"POST", "/path", false, ""},
	}
	processors := []reflect.StructTag{`path:"/path" method:"GET"`, `path:"/path/([^/]*?)" method:"GET"`, `path:"/path/([^/]*?)/([0-9]*)" method:"GET"`}
	funcs := []reflect.Method{mustGet(processor.MethodByName("handler1")), mustGet(processor.MethodByName("handler2")), mustGet(processor.MethodByName("handler3"))}

	for i, test := range tests {
		service := new(Service)
		err := initService(reflect.ValueOf(service).Elem(), `mime:"application/json" charset:"utf-8"`)
		if err != nil {
			t.Fatal(err)
		}
		for i, p := range processors {
			processor := new(Processor)
			err := initProcessor("/", reflect.ValueOf(processor).Elem(), p, funcs[i])
			if err != nil {
				t.Fatal(err)
			}
			service.processors = append(service.processors, *processor)
		}
		req, _ := http.NewRequest(test.method, fmt.Sprintf("http://127.0.0.1%s", test.url), nil)
		processor, ok := service.innerService.findProcessor(req)
		assert.Equal(t, ok, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			continue
		} else if !ok {
			t.Errorf("not find test %d", i)
			continue
		}
		assert.Equal(t, processor.path.String(), test.result, fmt.Sprintf("test %d", i))
	}
}
