package rest

import (
	"net/http"
	"testing"
)

type PreTest1 struct{}

func (t PreTest1) PreProcessor(r *http.Request, s Service, p Processor) *Response {
	return &Response{
		Status: 404,
	}
}

func (t PreTest1) Response(r *Response, s Service, p Processor) {}

type PreTest2 struct{}

func (t PreTest2) PreProcessor(r *http.Request, s Service, p Processor) *Response {
	r.Header.Set("Pre", "processed")
	return nil
}

func (t PreTest2) Response(r *Response, s Service, p Processor) {
	r.Header.Set("Post", "processed")
}

type PluginTest struct {
	Service

	Tester Processor `method:"GET" path:"/test"`

	t *testing.T
}

func (t PluginTest) Tester_() {
	if t.Request().Header.Get("Pre") != "processed" {
		t.t.Errorf("PreTest2.PreProcessor doesn't run")
	}
}

func TestPlugin(t *testing.T) {
	test := new(PluginTest)
	test.t = t

	{
		handler, err := New(test)
		if err != nil {
			t.Fatalf("create test error: %s", err)
		}
		handler.AddPlugin(new(PreTest1))

		req, _ := http.NewRequest("GET", "http://localhost/test", nil)
		_, code, _ := sendRequest(handler, req)
		if err != nil {
			t.Errorf("call failed: %s", err)
		}
		if code != 404 {
			t.Errorf("PreTest1.PreProcessor run error, return code: %d", code)
		}
	}

	{
		handler, err := New(test)
		if err != nil {
			t.Fatalf("create test error: %s", err)
		}
		handler.AddPlugin(new(PreTest2))

		req, _ := http.NewRequest("GET", "http://localhost/test", nil)
		_, code, header := sendRequest(handler, req)
		if err != nil {
			t.Errorf("call failed: %s", err)
		}
		if code != 200 {
			t.Errorf("run error, return code: %s", code)
		}
		if header.Get("Post") != "processed" {
			t.Errorf("PreTest2.Response doesn't run")
		}
	}
}
