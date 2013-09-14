package rest

import (
	"github.com/googollee/go-assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeTestHandler struct {
	pi *int
	i  int
}

func (h fakeTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	if h.pi != nil {
		*h.pi = h.i
	}
}

func TestEndPoint(t *testing.T) {
	call := 0
	h1 := fakeTestHandler{&call, 1}
	h2 := fakeTestHandler{&call, 2}
	type Test struct {
		method string
		f      Handler
		ok     bool
		called int
		target string
	}
	var tests = []Test{
		{"", h1, false, 0, ""},
		{"GET", nil, false, 0, ""},
		{"GET", h1, true, 1, "GET"},
		{"POST", h1, true, 1, "GET, POST"},
		{"GET", h2, false, 0, "GET, POST"},
		{"HEAD", h2, true, 2, "GET, POST, HEAD"},
		{"Custom1", h2, true, 2, "GET, POST, HEAD, Custom1"},
		{"Custom1", h2, false, 2, "GET, POST, HEAD, Custom1"},
		{"Custom2", h2, true, 2, "GET, POST, HEAD, Custom1, Custom2"},
	}
	ep := NewEndPoint()
	for i, test := range tests {
		err := ep.Add(test.method, test.f)
		assert.MustEqual(t, err == nil, test.ok, "test %d", i)
		if err != nil {
			continue
		}
		assert.Equal(t, ep.Methods(), test.target, "test %d", i)
	}
}

func TestEndPointCall(t *testing.T) {
	call := 0
	h1 := fakeTestHandler{&call, 1}
	h2 := fakeTestHandler{&call, 2}
	ep := NewEndPoint()
	ep.Add("GET", h1)
	ep.Add("POST", h2)
	ep.Add("Custom1", h1)
	ep.Add("Custom2", h2)
	type Test struct {
		method string
		url    string
		code   int
		called int
	}
	var tests = []Test{
		{"GET", "http://domain/path", http.StatusOK, 1},
		{"POST", "http://domain/path", http.StatusOK, 2},
		{"Custom1", "http://domain/path", http.StatusOK, 1},
		{"Custom2", "http://domain/path", http.StatusOK, 2},
		{"OPTIONS", "http://domain/path", http.StatusMethodNotAllowed, 0},
		{"PUT", "http://domain/path", http.StatusMethodNotAllowed, 0},
		{"NotExist", "http://domain/path", http.StatusMethodNotAllowed, 0},
		{"GET", "http://domain/path?_method=Custom1", http.StatusOK, 1},
		{"GET", "http://domain/path?_method=NotExist", http.StatusMethodNotAllowed, 0},
	}
	for i, test := range tests {
		req, err := http.NewRequest(test.method, test.url, nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		resp := httptest.NewRecorder()
		resp.Code = http.StatusOK
		call = 0
		ep.Call(resp, req, nil)
		assert.Equal(t, resp.Code, test.code, "test %d", i)
		if resp.Code != http.StatusOK {
			assert.Equal(t, resp.Header().Get("Allow"), ep.Methods(), "test %d", i)
		}
		assert.Equal(t, call, test.called, "test %d", i)
	}
}

func TestEndPointCallConcurrency(t *testing.T) {
	h1 := fakeTestHandler{}
	h2 := fakeTestHandler{}
	ep := NewEndPoint()
	ep.Add("GET", h1)
	ep.Add("POST", h2)
	ep.Add("Custom1", h1)
	ep.Add("Custom2", h2)
	n := 100
	quit := make(chan int)
	for i := 0; i < n; i++ {
		go func(i int) {
			req, err := http.NewRequest("GET", "http://domain/path", nil)
			assert.MustEqual(t, err, nil, "test %d", i)
			resp := httptest.NewRecorder()
			resp.Code = http.StatusOK
			ep.Call(resp, req, nil)
			assert.Equal(t, resp.Code, http.StatusOK, "test %d", i)
			quit <- 1
		}(i)
	}
	for i := 0; i < n; i++ {
		<-quit
	}
	close(quit)
}
