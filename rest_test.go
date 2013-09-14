package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
)

type fakeFailRest struct {
	Service

	notExist FakeNode
}

type fakeDupRest struct {
	Service

	handler1 FakeNode
	handler2 FakeNode
}

func (r *fakeDupRest) Handler1() {}
func (r *fakeDupRest) Handler2() {}

type fakeDupServiceRest struct {
	S1 Service `prefix:"/prefix"`
	S2 Service `prefix:"/prefix"`

	handler1 FakeNode `route:"/handler1"`
	handler2 FakeNode `route:"/handerl2"`
}

func (r *fakeDupServiceRest) Handler1() {}
func (r *fakeDupServiceRest) Handler2() {}

type fakeDoubleServiceRest struct {
	S1 Service `prefix:"/prefix1"`
	S2 Service `prefix:"/prefix2"`

	handler1 FakeNode `route:"/handler1"`
	handler2 FakeNode `route:"/handerl2"`
}

func (r *fakeDoubleServiceRest) Handler1() {}
func (r *fakeDoubleServiceRest) Handler2() {}

type fakeRest struct {
	Service `prefix:"/prefix"`

	handler1 FakeNode `route:"/handler1"`
	handler2 FakeNode `route:"/handler2"`

	LastCall string
}

func (r *fakeRest) Handler1() { r.LastCall = "handler1" }
func (r *fakeRest) Handler2() { r.LastCall = "handler2" }

func TestRestAdd(t *testing.T) {
	type Test struct {
		serviceTag reflect.StructTag
		v          interface{}
		ok         bool
		length     int
		paths      string
	}
	var tests = []Test{
		{`prefix:"/prefix"`, 1, false, 0, ""},
		{`prefix:"/prefix"`, "", false, 0, ""},
		{`prefix:"/prefix"`, new(int), false, 0, ""},
		{`prefix:"/prefix"`, new(fakeFailRest), false, 0, ""},
		{`prefix:"/prefix"`, new(fakeDupRest), false, 0, ""},
		{`prefix:"/prefix"`, new(fakeDupServiceRest), false, 0, ""},
		{`prefix:"/prefix"`, new(fakeRest), true, 2, "[/prefix/handler1 /prefix/handler2]"},
		{`prefix:"/prefix"`, new(fakeDoubleServiceRest), true, 4, "[/prefix1/handerl2 /prefix1/handler1 /prefix2/handerl2 /prefix2/handler1]"},
	}
	for i, test := range tests {
		rest := New()
		err := rest.Add(test.v)
		assert.Equal(t, err == nil, test.ok, "test %d error: %s", i, err)
		if err != nil {
			continue
		}
		assert.Equal(t, len(rest.router.Routes), test.length, "test %d", i)
		var paths []string
		for _, path := range rest.router.Routes {
			paths = append(paths, path.PathExp)
			_, ok := path.Dest.(*EndPoint)
			assert.MustEqual(t, ok, true, "test %d")

		}
		sort.Strings(paths)
		assert.Equal(t, fmt.Sprintf("%v", paths), test.paths, "test %d", i)
	}
}

func TestRestServeHTTP(t *testing.T) {
	type Test struct {
		method string
		url    string
		code   int
		call   string
	}
	var tests = []Test{
		{"GET", "http://domain/prefix/handler1", http.StatusMethodNotAllowed, ""},
		{"GET", "http://domain/non/exist", http.StatusNotFound, ""},
		{"FAKE_METHOD", "http://domain/prefix/handler1", http.StatusOK, "handler1"},
		{"FAKE_METHOD", "http://domain/prefix/handler2", http.StatusOK, "handler2"},
	}
	rest := New()
	service := new(fakeRest)
	err := rest.Add(service)
	assert.MustEqual(t, err, nil, "error: %s", err)
	var handler http.Handler
	handler = rest
	for i, test := range tests {
		req, err := http.NewRequest(test.method, test.url, nil)
		assert.MustEqual(t, err, nil, "test %d error: %s", i, err)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, test.code, "test %d", i)
		assert.Equal(t, service.LastCall, test.call, "test %d", i)
	}
}
