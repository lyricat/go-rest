package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"
)

var ret = "Hello world."

type BenchmarkRest struct {
	Service `prefix:"/prefix"`

	Fake FakeNode  `method:"GET" path:"/fake/:id" func:"HandleGet"`
	Get  Processor `method:"GET" path:"/processor/:id"`
	Post Processor `method:"POST" path:"/processor/:id/post"`
	Full Processor `method:"POST" path:"/processor/:id/full"`
}

func (r BenchmarkRest) HandleGet() string {
	return ret
}

func (r BenchmarkRest) HandlePost(arg string) {}

func (r BenchmarkRest) HandleFull(arg string) string {
	return arg
}

var rest *Rest

func init() {
	var err error
	rest, err = New(new(BenchmarkRest))
	if err != nil {
		panic(err)
	}
}

func BenchmarkRestServe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", "http://127.0.0.1/prefix/fake/id", nil)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		rest.ServeHTTP(resp, req)
	}
}

func BenchmarkRestGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", "http://127.0.0.1/prefix/processor/id", nil)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		rest.ServeHTTP(resp, req)
	}
}

func BenchmarkRestPost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBufferString("\"post\"")
		req, err := http.NewRequest("POST", "http://127.0.0.1/prefix/processor/id/post", buf)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		rest.ServeHTTP(resp, req)
	}
}

func BenchmarkRestFull(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBufferString("\"post\"")
		req, err := http.NewRequest("POST", "http://127.0.0.1/prefix/processor/id/full", buf)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		rest.ServeHTTP(resp, req)
	}
}

var handlers = []struct {
	path    *regexp.Regexp
	handler http.HandlerFunc
}{
	{regexp.MustCompile("^/prefix/processor/(.*?)$"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		encoder := json.NewEncoder(w)
		err := encoder.Encode(ret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}},
	{regexp.MustCompile("^/prefix/processor/(.*?)$/post"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var arg string
		err := decoder.Decode(&arg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}},
	{regexp.MustCompile("^/prefix/processor/(.*?)$/full"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var arg string
		err := decoder.Decode(&arg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		encoder := json.NewEncoder(w)
		err = encoder.Encode(arg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}},
}

func BenchmarkPlainGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", "http://127.0.0.1/prefix/processor/id", nil)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		for _, h := range handlers {
			if len(h.path.FindAllStringSubmatch(req.URL.Path, -1)) > 0 {
				h.handler(resp, req)
			}
		}
	}
}

func BenchmarkPlainPost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBufferString("\"post\"")
		req, err := http.NewRequest("GET", "http://127.0.0.1/prefix/processor/id", buf)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		for _, h := range handlers {
			if len(h.path.FindAllStringSubmatch(req.URL.Path, -1)) > 0 {
				h.handler(resp, req)
			}
		}
	}
}

func BenchmarkPlainFull(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBufferString("\"post\"")
		req, err := http.NewRequest("GET", "http://127.0.0.1/prefix/processor/id", buf)
		if err != nil {
			panic(err)
		}
		resp := newWriter()
		for _, h := range handlers {
			if len(h.path.FindAllStringSubmatch(req.URL.Path, -1)) > 0 {
				h.handler(resp, req)
			}
		}
	}
}
