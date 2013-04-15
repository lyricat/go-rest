package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
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
		{"prefix", "path", nil, "/prefix/path", "/prefix/path"},
		{"/prefix", "/path", nil, "/prefix/path", "/prefix/path"},
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

type innerFakeNode struct {
	formatter pathFormatter
	f         reflect.Method
	tag       reflect.StructTag

	lastInstance reflect.Value
	lastCtx      *context
}

func (h *innerFakeNode) init(formatter pathFormatter, f reflect.Method, tag reflect.StructTag) error {
	h.formatter = formatter
	h.f = f
	h.tag = tag
	return nil
}

func (h *innerFakeNode) handle(instance reflect.Value, ctx *context) {
	h.lastInstance = instance
	h.lastCtx = ctx
}

type FakeNode struct {
	*innerFakeNode
}

type FakeNodeService struct {
	FNode     FakeNode `method:"m" path:"p" func:"F" other:"o"`
	Node      FakeNode `method:"m" path:"p" other:"o"`
	NoStruct  int
	NoPtr     struct{ A int }
	NoHandler struct{ A *int }
	NoMethod  FakeNode `path:"p" other:"o"`
	NoPath    FakeNode `method:"m" other:"o"`
}

func (s FakeNodeService) F()          {}
func (s FakeNodeService) HandleNode() {}

func TestNewNode(t *testing.T) {
	s := new(FakeNodeService)
	instance := reflect.ValueOf(s)
	type Test struct {
		instance reflect.Value
		node     string
		prefix   string

		ok           bool
		invalidError bool
		method       string
		f            string
		formatter    pathFormatter
		tag          reflect.StructTag
	}
	var tests = []Test{
		{instance, "FNode", "", true, false, "m", "F", "/p", `method:"m" path:"p" func:"F" other:"o"`},
		{instance, "Node", "", true, false, "m", "HandleNode", "/p", `method:"m" path:"p" other:"o"`},
		{instance, "NoStruct", "", false, true, "", "", "", ""},
		{instance, "NoPtr", "", false, true, "", "", "", ""},
		{instance, "NoHandler", "", false, true, "", "", "", ""},
		{instance, "NoMethod", "", false, false, "", "", "", ""},
		{instance, "NoPath", "", false, false, "", "", "", ""},
	}

	s.FNode.innerFakeNode = nil
	s.Node.innerFakeNode = nil
	for i, test := range tests {
		handler := test.instance.Elem().FieldByName(test.node)
		nodeType, ok := test.instance.Elem().Type().FieldByName(test.node)
		if !ok {
			t.Fatalf("can't find %s", test.node)
		}
		n, formatter, err := newNode(test.instance.Elem().Type(), handler, test.prefix, nodeType)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			assert.Equal(t, err == invalidHandler, test.invalidError, "test %d error: %s", i, err)
			continue
		}
		assert.Equal(t, formatter, test.formatter, fmt.Sprintf("test %d", i))
		assert.Equal(t, n.method, test.method, fmt.Sprintf("test %d", i))
		h := n.handler.(*innerFakeNode)
		assert.Equal(t, h.formatter, test.formatter, fmt.Sprintf("test %d", i))
		assert.Equal(t, h.tag, test.tag, fmt.Sprintf("test %d", i))
		assert.Equal(t, h.f.Name, test.f, fmt.Sprintf("test %d", i))
	}
	assert.NotEqual(t, s.FNode.innerFakeNode, nil)
	assert.NotEqual(t, s.Node.innerFakeNode, nil)
}
