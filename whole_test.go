package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type WholeTest struct {
	Service `prefix:"/prefix"`

	handler1    SimpleNode `route:"/handler1/:id" method:"GET"`
	handler2    SimpleNode `path:"/handler2/:id" method:"POST"`
	concurrency SimpleNode `path:"/concurrency/:id" method:"PUT"`

	streaming Streaming `path:"/stream" method:"WATCH"`

	t        *testing.T
	lastCall string
	p        chan int
	r        chan int
}

func (w *WholeTest) Handler1(ctx Context, i int) {
	var id int
	ctx.Bind("id", &id)
	if err := ctx.BindError(); err != nil {
		ctx.Return(http.StatusBadRequest, err)
		return
	}
	w.lastCall = fmt.Sprintf("call handler1 with %d, id %d", i, id)
	ctx.Response().Header().Set("Return-Header", "return")
	ctx.Return(http.StatusAccepted)
	ctx.Render("str")
}

func (w *WholeTest) Handler2(ctx Context, s *string) {
	var id string
	ctx.Bind("id", &id)
	if err := ctx.BindError(); err != nil {
		ctx.Return(http.StatusBadRequest, err)
		return
	}
	header := ctx.Request().Header.Get("Custom-Header")
	w.lastCall = fmt.Sprintf("call handler2 with %s, header %s, id %s", *s, header, id)

	ctx.Render(1)
}

func (w *WholeTest) Concurrency(ctx Context, i int) {
	var id int
	ctx.Bind("id", &id)
	if err := ctx.BindError(); err != nil {
		ctx.Return(http.StatusBadRequest, err)
		return
	}
	ctx.Render(fmt.Sprintf("id: %d, i: %d", id, i))
}

func (w *WholeTest) Streaming(ctx StreamContext) {
	ctx.Response().Header().Set("Return-Header", "return")
	ctx.Return(http.StatusAccepted)
	err := ctx.Render(1)
	assert.MustEqual(w.t, err, nil)
	w.p <- 1
	<-w.r
	err = ctx.Render(2)
	assert.MustEqual(w.t, err, nil)
	w.p <- 1
	return
}

func TestWholeRest(t *testing.T) {
	rest := &WholeTest{
		t: t,
		p: make(chan int),
		r: make(chan int),
	}
	s := New()
	err := s.Add(rest)
	assert.MustEqual(t, err, nil)
	server := httptest.NewServer(s)
	defer server.Close()

	{
		req, err := http.NewRequest("GET", server.URL+"/prefix/handler1/123", strings.NewReader("1"))
		assert.MustEqual(t, err, nil)
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, err, nil)
		assert.Equal(t, resp.StatusCode, http.StatusAccepted)
		b, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, err, nil)
		assert.Equal(t, string(b), "\"str\"\n")
		assert.Equal(t, rest.lastCall, "call handler1 with 1, id 123")
		assert.Equal(t, resp.Header.Get("Return-Header"), "return")
		assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")
	}

	{
		req, err := http.NewRequest("GET", server.URL+"/prefix/handler1/abc", strings.NewReader("1"))
		assert.MustEqual(t, err, nil)
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, err, nil)
		assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
		b, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, err, nil)
		assert.Equal(t, string(b), "id(id)'s value(abc) is invalid int\n")
	}

	{
		req, err := http.NewRequest("POST", server.URL+"/handler2/abc", strings.NewReader("\"str\""))
		assert.MustEqual(t, err, nil)
		req.Header.Set("Custom-Header", "custom")
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, err, nil)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
		b, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, err, nil)
		assert.Equal(t, string(b), "1\n")
		assert.Equal(t, rest.lastCall, "call handler2 with str, header custom, id abc")
		assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")
	}

	{
		req, err := http.NewRequest("GET", server.URL+"/handler2/abc?_method=POST", strings.NewReader("\"str\""))
		assert.MustEqual(t, err, nil)
		req.Header.Set("Custom-Header", "custom")
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, err, nil)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
		b, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, err, nil)
		assert.Equal(t, string(b), "1\n")
		assert.Equal(t, rest.lastCall, "call handler2 with str, header custom, id abc")
		assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")
	}

	{
		req, err := http.NewRequest("GET", server.URL+"/handler2/abc", strings.NewReader("\"str\""))
		assert.MustEqual(t, err, nil)
		req.Header.Set("Custom-Header", "custom")
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, err, nil)
		assert.Equal(t, resp.StatusCode, http.StatusMethodNotAllowed)
		assert.Equal(t, resp.Header["Allow"], []string{"POST"})
	}

	{
		req, err := http.NewRequest("WATCH", server.URL+"/stream", nil)
		assert.MustEqual(t, err, nil)
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, err, nil)
		assert.Equal(t, resp.StatusCode, http.StatusAccepted)
		<-rest.p
		b := make([]byte, 1024)
		n, err := resp.Body.Read(b)
		assert.Equal(t, err, nil)
		assert.Equal(t, string(b[:n]), "1\n")
		rest.r <- 1
		<-rest.p
		n, err = resp.Body.Read(b)
		assert.Equal(t, err, nil)
		assert.Equal(t, string(b[:n]), "2\n")
	}
}

func TestWholeRestConcurrency(t *testing.T) {
	rest := &WholeTest{
		t: t,
		p: make(chan int),
		r: make(chan int),
	}
	s := New()
	err := s.Add(rest)
	assert.MustEqual(t, err, nil)
	server := httptest.NewServer(s)
	defer server.Close()

	n := 100
	quit := make(chan int)
	for i := 0; i < n; i++ {
		go func() {
			req, err := http.NewRequest("PUT", server.URL+"/concurrency/1", strings.NewReader("1"))
			assert.Equal(t, err, nil, "error: %s", err)
			resp, err := http.DefaultClient.Do(req)
			assert.Equal(t, err, nil, "error: %s", err)
			if err == nil {
				resp.Body.Close()
			}
			quit <- 1
		}()
	}
	for i := 0; i < n; i++ {
		<-quit
	}
}
