package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"net/http"
	"reflect"
	"sort"
	"testing"
)

type FakeHandler struct {
	serviceTag reflect.StructTag
	fieldTag   reflect.StructTag
	fname      string
	f          string
	method     reflect.Value
}

func (h FakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	h.method.Call(nil)
}

func (h FakeHandler) String() string {
	return fmt.Sprintf("stag: %s, ftag: %s, fname: %s, f: %s", h.serviceTag, h.fieldTag, h.fname, h.f)
}

type FakeNode struct{}

func (n FakeNode) CreateHandler(service reflect.StructTag, field reflect.StructTag, fname string, f reflect.Value) (path string, method string, handler Handler, err error) {
	if fname == "CreateFailed" {
		return "", "", nil, fmt.Errorf("failed")
	}
	return service.Get("prefix") + field.Get("route"), "FAKE_METHOD", FakeHandler{
		serviceTag: service,
		fieldTag:   field,
		fname:      fname,
		f:          f.Type().String(),
		method:     f,
	}, nil
}

type fakeNonexistService struct {
	nonexist FakeNode `route:"/nonexist"`
}

type fakeFailedService struct {
	createFailed FakeNode `route:"/create/failed"`
}

func (s *fakeFailedService) CreateFailed() {}

type fakeDupService struct {
	handler1 FakeNode `route:"/handler1"`
	handler2 FakeNode `route:"/handler1"`
}

func (s *fakeDupService) Handler1(int)    {}
func (s *fakeDupService) Handler2(string) {}

type fakeEmptyService struct{}

type fakeOKService struct {
	handler1 FakeNode `route:"/handler1"`
	handler2 FakeNode `route:"/handler2"`
}

func (s *fakeOKService) Handler1(int)    {}
func (s *fakeOKService) Handler2(string) {}

func TestService(t *testing.T) {
	s := Service{}
	type Test struct {
		serviceTag reflect.StructTag
		v          interface{}

		ok       bool
		length   int
		handlers string
	}
	var tests = []Test{
		{"", new(fakeNonexistService), false, 0, ""},
		{"", new(fakeFailedService), false, 0, ""},
		{"", new(fakeDupService), false, 0, ""},
		{"", new(fakeEmptyService), true, 0, ""},
		{`prefix:"/prefix"`, new(fakeOKService), true, 2, ",|FAKE_METHOD,stag: prefix:\"/prefix\", ftag: route:\"/handler1\", fname: Handler1, f: func(int),|FAKE_METHOD,stag: prefix:\"/prefix\", ftag: route:\"/handler2\", fname: Handler2, f: func(string)"},
	}
	for i, test := range tests {
		handlers, err := s.MakeHandlers(test.serviceTag, test.v)
		assert.MustEqual(t, err == nil, test.ok, "test %d: %s", i, err)
		if err != nil {
			continue
		}
		assert.Equal(t, len(handlers), test.length, "test %d", i)
		var paths []string
		for path := range handlers {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		str := ""
		for _, path := range paths {
			p := ""
			for method, f := range handlers[path].funcs {
				p = fmt.Sprintf("%s|%s,%s", p, method, f)
			}
			str = fmt.Sprintf("%s,%s", str, p)
		}
		assert.Equal(t, str, test.handlers, "test %d", i)
	}
}
