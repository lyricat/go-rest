package rest

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
)

var invalidHandler = errors.New("invalid handler")

type pathFormatter string

func pathToFormatter(prefix, path string) pathFormatter {
	if len(prefix) == 0 || prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if len(path) > 0 {
		prefixLast := prefix[len(prefix)-1]
		if prefixLast != '/' && path[0] != '/' {
			prefix = prefix + "/"
		}
		if prefixLast == '/' && path[0] == '/' {
			path = path[1:]
		}
	}
	return pathFormatter(prefix + path)
}

func (f pathFormatter) pathMap(args map[string]string) string {
	ret := string(f)
	for k, v := range args {
		ret = strings.Replace(ret, ":"+k, v, -1)
	}
	return ret
}

func (f pathFormatter) path(params ...string) string {
	var key string
	m := make(map[string]string)
	for i, p := range params {
		if i&1 == 0 {
			key = p
		} else {
			m[key] = p
			key = ""
		}
	}
	if key != "" {
		m[key] = ""
	}
	return f.pathMap(m)
}

type node interface {
	init(formatter pathFormatter, instance reflect.Type, name string, tag reflect.StructTag) ([]handler, []pathFormatter, error)
}

type handler interface {
	handle(instance reflect.Value, ctx *context)
}

type processorNode struct {
	funcIndex    int
	requestType  reflect.Type
	responseType reflect.Type
}

func (n *processorNode) handle(instance reflect.Value, ctx *context) {
	r := ctx.request
	w := ctx.responseWriter
	marshaller := ctx.marshaller
	f := instance.Method(n.funcIndex)
	var args []reflect.Value

	if n.requestType != nil {
		request := reflect.New(n.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		args = append(args, request.Elem())
	}
	ret := f.Call(args)

	if !ctx.isError && len(ret) > 0 {
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=%s", ctx.mime, ctx.charset))
		err := marshaller.Marshal(w, ret[0].Interface())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type streamingWriter struct {
	conn         net.Conn
	bufrw        *bufio.ReadWriter
	header       http.Header
	writedHeader bool
}

func (w *streamingWriter) Write(b []byte) (int, error) {
	if !w.writedHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(b)
}

func (w *streamingWriter) Header() http.Header {
	return w.header
}

func (w *streamingWriter) WriteHeader(code int) {
	if w.writedHeader {
		return
	}
	w.bufrw.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", code, http.StatusText(code))))
	w.Header().Write(w.bufrw)
	w.bufrw.Write([]byte("\r\n"))
	w.bufrw.Flush()
	w.writedHeader = true
}

type streamingNode struct {
	funcIndex   int
	end         string
	requestType reflect.Type
}

func (n *streamingNode) handle(instance reflect.Value, ctx *context) {
	r := ctx.request
	w := ctx.responseWriter
	f := instance.Method(n.funcIndex)
	marshaller := ctx.marshaller

	var request reflect.Value
	if n.requestType != nil {
		request = reflect.New(n.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		request = reflect.Indirect(request)
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ctx.responseWriter = &streamingWriter{
		conn:         conn,
		bufrw:        bufrw,
		header:       make(http.Header),
		writedHeader: false,
	}

	stream := newStream(ctx, conn, n.end)

	args := []reflect.Value{reflect.ValueOf(stream).Elem()}
	if n.requestType != nil {
		args = append(args, request)
	}

	ctx.responseWriter.Header().Set("Connection", "keep-alive")
	ctx.responseWriter.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", ctx.mime))

	f.Call(args)
}
