package rest

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type FakeNode struct {
	formatter    pathFormatter
	lastInstance reflect.Value
	lastCtx      *context
}

func (n *FakeNode) init(formatter pathFormatter, instance reflect.Value, name string, tag reflect.StructTag) ([]handler, []pathFormatter, error) {
	n.formatter = formatter
	return []handler{&FakeHandler{name, n}}, []pathFormatter{formatter}, nil
}

type FakeHandler struct {
	name_ string
	node  *FakeNode
}

func (h *FakeHandler) name() string {
	return h.name_
}

func (h *FakeHandler) handle(instance reflect.Value, ctx *context) {
	h.node.lastInstance = instance
	h.node.lastCtx = ctx
}

type TestDefault struct {
	Service `prefix:"/prefix" mime:"mime" charset:"charset"`

	NoMethod FakeNode `path:"/default" method:"METHOD" other:"other"`
}

type TestFunc struct {
	NoMethod FakeNode `path:"/func" method:"METHOD" func:"FuncHandler"`

	Service `prefix:"/prefix" mime:"mime" charset:"charset"`
}

type TestNoMethod struct {
	Service `prefix:"/prefix" mime:"mime" charset:"charset"`

	NoMethod FakeNode `path:"/no/method"`
}

type TestNoPath struct {
	Service `prefix:"/prefix" mime:"mime" charset:"charset"`

	NoMethod FakeNode `method:"METHOD"`
}

type TestSamePath struct {
	Service `prefix:"/prefix" mime:"mime" charset:"charset"`

	NoMethod1 FakeNode `method:"METHOD"`
	NoMethod2 FakeNode `method:"METHOD"`
}

type TestNoService struct{}

func TestNewRest(t *testing.T) {
	type Test struct {
		instance interface{}

		ok           bool
		serviceIndex int
		prefix       string
		mime         string
		charset      string
		formatter    pathFormatter
		tag          reflect.StructTag
	}
	var tests = []Test{
		{new(TestDefault), true, 0, "/prefix/", "mime", "charset", "/prefix/default", `path:"/default" method:"METHOD" other:"other"`},
		{new(TestFunc), true, 1, "/prefix/", "mime", "charset", "/prefix/func", `path:"/func" method:"METHOD" func:"FuncHandler"`},
		{new(TestNoPath), true, 0, "/prefix/", "mime", "charset", "/prefix/", `method:"METHOD"`},
		{new(TestNoService), false, 0, "", "", "", "", ""},
		{new(TestNoMethod), false, 0, "", "", "", "", ""},
		{new(TestSamePath), false, 0, "", "", "", "", ""},
	}
	for i, test := range tests {
		r, err := New(test.instance)
		equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if err != nil || !test.ok {
			continue
		}
		equal(t, r.Prefix(), test.prefix, fmt.Sprintf("test %d"), i)
		equal(t, r.defaultMime, test.mime, fmt.Sprintf("test %d"), i)
		equal(t, r.defaultCharset, test.charset, fmt.Sprintf("test %d"), i)
		handler, ok := r.router.Routes[0].Dest.(*FakeHandler)
		if !ok {
			fmt.Errorf("handler not *FakeHandler")
			continue
		}
		equal(t, handler.node.formatter, test.formatter, fmt.Sprintf("test %d", i))
	}
}

type TestPost struct {
	Service `prefix:"/prefix"`

	Node   FakeNode `method:"POST" path:"/node"`
	NodeId FakeNode `method:"GET" path:"/node/:id" func:"HandleNode"`
}

func (r TestPost) HandleNode() {}

func TestRestServeHTTP(t *testing.T) {
	type Test struct {
		method string
		url    string

		code      int
		node      *FakeNode
		formatter pathFormatter
		vars      map[string]string
	}
	instance := new(TestPost)
	rest, err := New(instance)
	if err != nil {
		t.Fatalf("new rest service failed: %s", err)
	}
	var tests = []Test{
		{"GET", "http://domain/prefix/node/123", http.StatusOK, &instance.NodeId, "/prefix/node/:id", map[string]string{"id": "123"}},
		{"GET", "http://domain/prefix/node/", http.StatusNotFound, nil, "", nil},

		{"POST", "http://domain/prefix/node", http.StatusOK, &instance.Node, "/prefix/node", nil},
		{"POST", "http://domain/prefix/no/exist", http.StatusNotFound, nil, "", nil},
		{"GET", "http://domain/prefix/node", http.StatusNotFound, nil, "", nil},
	}
	for i, test := range tests {
		buf := bytes.NewBuffer(nil)
		req, err := http.NewRequest(test.method, test.url, buf)
		if err != nil {
			t.Fatalf("test %d create request failed", i, err)
		}
		w := httptest.NewRecorder()
		w.Code = http.StatusOK
		rest.ServeHTTP(w, req)
		equal(t, w.Code, test.code, fmt.Sprintf("test %d code: %s", i, w.Code))
		if w.Code != http.StatusOK {
			continue
		}
		equal(t, test.node.formatter, test.formatter, fmt.Sprintf("test %d", i))
		equal(t, equalMap(test.node.lastCtx.vars, test.vars), true, fmt.Sprintf("test %d", i))

		service := test.node.lastInstance.Field(0).Interface().(Service)
		equal(t, equalMap(service.Vars(), test.vars), true, fmt.Sprintf("test %d", i))
	}
}
