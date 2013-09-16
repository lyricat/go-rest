package rest

import (
	"fmt"
	"net/http"
	"reflect"
)

// SimpleNode is node to make http handler. It's tag has below parameters :
//  - method: http request method which need be handled.
//  - route: service's tag prefix add route is http request url path.
//  - path: path will ignore service's prefix tag, and use as url path.
type SimpleNode struct{}

// CreateHandler will create a set of handlers.
func (p SimpleNode) CreateHandler(serviceTag reflect.StructTag, fieldTag reflect.StructTag, fname string, f reflect.Value) (string, string, Handler, error) {
	path := fieldTag.Get("path")
	if path == "" {
		path = serviceTag.Get("prefix") + fieldTag.Get("route")
	}
	if path == "" {
		path = "/"
	}

	method := fieldTag.Get("method")
	if len(method) == 0 {
		return "", "", nil, fmt.Errorf("method should NOT be empty")
	}

	mime := serviceTag.Get("mime")
	marshaller, ok := getMarshaller(mime)
	if !ok {
		mime, marshaller = "application/json", jsonMarshaller
	}

	t := f.Type()
	if t.NumIn() != 1 && t.NumIn() != 2 {
		return "", "", nil, fmt.Errorf("handler method %s should have 1 or 2 input parameters", fname)
	}
	p0 := t.In(0)
	if p0.Kind() != reflect.Interface || p0.Name() != "Context" {
		return "", "", nil, fmt.Errorf("handler method %s's 1st parameter must be rest.Context", fname)
	}
	var p1 reflect.Type
	if t.NumIn() == 2 {
		p1 = t.In(1)
	}

	return path, method, &baseHandler{fname, mime, marshaller, p1, f}, nil
}

type baseHandler struct {
	name       string
	mime       string
	marshaller Marshaller
	inputType  reflect.Type
	f          reflect.Value
}

func (h *baseHandler) Name() string {
	return h.name
}

func (h *baseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	mime, marshaller := getMarshallerFromRequest(h.mime, h.marshaller, r)

	ctx := newBaseContext(h.name, marshaller, "utf-8", vars, r, w)
	ctx.Response().Header().Set("Content-Type", mime)

	args := []reflect.Value{reflect.ValueOf(ctx)}
	if h.inputType != nil {
		arg, err := unmarshallFromReader(h.inputType, marshaller, r.Body)
		if err != nil {
			ctx.Return(http.StatusBadRequest, "decode request body error: %s", err)
			return
		}
		args = append(args, arg)
	}

	h.f.Call(args)
}
