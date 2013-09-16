package rest

import (
	"github.com/googollee/go-assert"
	"testing"
)

func TestCheckRoute(t *testing.T) {
	rest := &WholeTest{
		t: t,
		p: make(chan int),
		r: make(chan int),
	}
	s := New()
	err := s.Add(rest)
	assert.MustEqual(t, err, nil)
	type Test struct {
		path   string
		method string
		ok     bool
		name   string
		vars   map[string]string
	}
	var tests = []Test{
		{"/prefix/handler1/123", "GET", true, "Handler1", map[string]string{"id": "123"}},
		{"/handler2/123", "POST", true, "Handler2", map[string]string{"id": "123"}},
		{"/concurrency/123", "PUT", true, "Concurrency", map[string]string{"id": "123"}},
		{"/stream", "WATCH", true, "Streaming", map[string]string{}},

		{"/streaming", "WATCH", false, "", nil},
		{"/stream", "GET", false, "", nil},
		{"/prefix/handler1", "GET", false, "", nil},
	}
	for i, test := range tests {
		name, vars, err := CheckRoute(s, test.path, test.method)
		assert.MustEqual(t, err == nil, test.ok, "test %d, error: %s", i, err)
		if err != nil {
			continue
		}
		assert.Equal(t, name, test.name, "test %d", i)
		assert.Equal(t, vars, test.vars, "test %d", i)
	}
}
