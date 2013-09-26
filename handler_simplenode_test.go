package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type nodeFuncs struct {
	ctx    Context
	called string
}

func (f *nodeFuncs) Ctx(ctx Context) {
	f.ctx = ctx
	f.called = "called ctx"
}

func (f *nodeFuncs) CtxInt(ctx Context, i int) {
	f.ctx = ctx
	f.called = fmt.Sprintf("called ctxInt with %d", i)
}

func (f *nodeFuncs) CtxPString(ctx Context, ps *string) {
	f.ctx = ctx
	f.called = fmt.Sprintf("called ctxPString with %s", *ps)
}

func (f *nodeFuncs) CtxReturn(ctx Context) int {
	f.ctx = ctx
	f.called = fmt.Sprintf("called ctxReturn")
	return 1
}

func (f *nodeFuncs) CtxIntReturn(ctx Context, i int) int {
	f.ctx = ctx
	f.called = fmt.Sprintf("called ctxIntReturn with %d", i)
	return i
}

func (f *nodeFuncs) NoArg()                        {}
func (f *nodeFuncs) MoreArg(ctx Context, i, j int) {}
func (f *nodeFuncs) NoContext1(i int)              {}
func (f *nodeFuncs) NoContext2(i, j int)           {}

func TestSimpleNode(t *testing.T) {
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
	f := new(nodeFuncs)
	var tests = []Test{
		{`mime:"application/json"`, `method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{`mime:"application/json" prefix:"/prefix"`, `method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/prefix", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{`prefix:"/prefix"`, `route:"/route" method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/prefix/route", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{`prefix:"/prefix"`, `path:"/path" method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/path", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{``, `method:"GET"`, "Ctx", reflect.ValueOf(f.Ctx), true, "/", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{``, `method:"GET"`, "CtxInt", reflect.ValueOf(f.CtxInt), true, "/", "GET", "rest.JSONMarshaller{}", "int"},
		{``, `method:"GET"`, "CtxPString", reflect.ValueOf(f.CtxPString), true, "/", "GET", "rest.JSONMarshaller{}", "*string"},
		{``, `method:"GET"`, "CtxReturn", reflect.ValueOf(f.CtxReturn), true, "/", "GET", "rest.JSONMarshaller{}", "<nil>"},
		{``, `method:"GET"`, "CtxIntReturn", reflect.ValueOf(f.CtxIntReturn), true, "/", "GET", "rest.JSONMarshaller{}", "int"},

		{``, ``, "NoMethod", reflect.ValueOf(f.Ctx), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoArg", reflect.ValueOf(f.NoArg), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "MoreArg", reflect.ValueOf(f.MoreArg), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoContext1", reflect.ValueOf(f.NoContext1), false, "/", "", "<nil>", "<nil>"},
		{``, `method:"GET"`, "NoContext2", reflect.ValueOf(f.NoContext2), false, "/", "", "<nil>", "<nil>"},
	}
	for i, test := range tests {
		var p Node
		p = SimpleNode{}
		path, method, handler, err := p.CreateHandler(test.serviceTag, test.fieldTag, test.fname, test.f)
		assert.MustEqual(t, err == nil, test.ok, "test %d", i)
		if err != nil {
			continue
		}
		assert.Equal(t, path, test.path, "test %d", i)
		assert.Equal(t, method, test.method, "test %d", i)
		ph, ok := handler.(*baseHandler)
		assert.MustEqual(t, ok, true, "test %d", i)
		assert.Equal(t, ph.name, test.fname, "test %d", i)
		assert.Equal(t, fmt.Sprintf("%#v", ph.marshaller), test.marshaller, "test %d", i)
		assert.Equal(t, fmt.Sprintf("%v", ph.inputType), test.inputType, "test %d", i)
	}
}

func TestBaseHandler(t *testing.T) {
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
	f := new(nodeFuncs)
	fakeMarshaller := FakeMarshaller{}
	RegisterMarshaller("fake/mime", fakeMarshaller)
	str := ""
	var tests = []Test{
		{"Ctx", fakeMarshaller, reflect.TypeOf(nil), reflect.ValueOf(f.Ctx), map[string][]string{"Content-Type": []string{"application/json"}}, "", nil, "called ctx", jsonMarshaller},
		{"Ctx", jsonMarshaller, reflect.TypeOf(nil), reflect.ValueOf(f.Ctx), nil, "", map[string]string{"ab": "cd"}, "called ctx", jsonMarshaller},
		{"CtxInt", jsonMarshaller, reflect.TypeOf(1), reflect.ValueOf(f.CtxInt), nil, "1", nil, "called ctxInt with 1", jsonMarshaller},
		{"CtxPString", jsonMarshaller, reflect.TypeOf(&str), reflect.ValueOf(f.CtxPString), nil, `"str"`, nil, "called ctxPString with str", jsonMarshaller},
		{"CtxReturn", jsonMarshaller, reflect.TypeOf(nil), reflect.ValueOf(f.CtxReturn), nil, "", nil, "called ctxReturn", jsonMarshaller},
		{"CtxIntReturn", jsonMarshaller, reflect.TypeOf(1), reflect.ValueOf(f.CtxIntReturn), nil, "1", nil, "called ctxIntReturn with 1", jsonMarshaller},
	}
	for i, test := range tests {
		handler := &baseHandler{
			name:       test.name,
			marshaller: test.marshaller,
			inputType:  test.inputType,
			f:          test.f,
		}
		req, err := http.NewRequest("GET", "http://method", strings.NewReader(test.body))
		assert.MustEqual(t, err, nil, "test %d", i)
		req.Header = test.header
		resp := httptest.NewRecorder()
		resp.Code = http.StatusOK
		handler.ServeHTTP(resp, req, test.vars)
		assert.Equal(t, resp.Code, http.StatusOK, "test %d", i)
		assert.Equal(t, f.called, test.called, "test %d", i)
		ctx, ok := f.ctx.(*baseContext)
		assert.MustEqual(t, ok, true, "test %d", i)
		assert.Equal(t, ctx.handlerName, test.name, "test %d", i)
		assert.Equal(t, ctx.vars, test.vars, "test %d", i)
		assert.Equal(t, ctx.marshaller, test.targetMarshaller, "test %d", i)
		assert.Equal(t, ctx.request, req, "test %d", i)
		assert.Equal(t, ctx.response, resp, "test %d", i)
	}
}

func TestBaseHandlerFail(t *testing.T) {
	f := new(nodeFuncs)
	failMarshaller := FailMarshaller{}
	RegisterMarshaller("fail/mime", failMarshaller)
	handler := &baseHandler{
		name:       "name",
		marshaller: jsonMarshaller,
		inputType:  reflect.TypeOf(1),
		f:          reflect.ValueOf(f.CtxInt),
	}
	req, err := http.NewRequest("GET", "http://method", strings.NewReader("1"))
	assert.MustEqual(t, err, nil)
	req.Header.Set("Content-Type", "fail/mime")
	resp := httptest.NewRecorder()
	resp.Code = http.StatusOK
	handler.ServeHTTP(resp, req, nil)
	assert.Equal(t, resp.Code, http.StatusBadRequest)
}
