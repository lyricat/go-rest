package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

var ret = "Hello world."

type BenchmarkRest struct {
	Service `prefix:"/prefix"`

	full SimpleNode `method:"POST" path:"/processor/:id/full"`
}

func (r BenchmarkRest) Full(ctx Context, arg string) {
	var id string
	ctx.Bind("id", &id)
	if err := ctx.BindError(); err != nil {
		ctx.Return(http.StatusBadRequest, err)
		return
	}
	ctx.Render(id + ret)
}

var rest *Rest

func init() {
	rest = New()
	if err := rest.Add(new(BenchmarkRest)); err != nil {
		panic(err)
	}
}

func BenchmarkRestFull(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBufferString("\"post\"")
		req, err := http.NewRequest("POST", "http://127.0.0.1/prefix/processor/id/full", buf)
		if err != nil {
			panic(err)
		}
		resp := httptest.NewRecorder()
		rest.ServeHTTP(resp, req)
	}
}
