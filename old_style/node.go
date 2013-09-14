package rest

import (
	"errors"
	"fmt"
	"io"
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

// Generate the path of url to processor. Map args fill parameters in path.
func (f pathFormatter) PathMap(args map[string]string) string {
	ret := string(f)
	for k, v := range args {
		ret = strings.Replace(ret, ":"+k, v, -1)
	}
	return ret
}

// Generate the path of url to processor. It accepts a sequence of key/value pairs, and fill parameters in path.
func (f pathFormatter) Path(params ...string) string {
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
	return f.PathMap(m)
}

type node interface {
	init(formatter pathFormatter, instance reflect.Type, name string, tag reflect.StructTag) ([]handler, []pathFormatter, error)
}

type handler interface {
	name() string
	handle(instance reflect.Value, ctx *context)
}

type processorWriter struct {
	resp   http.ResponseWriter
	writer io.Writer
}

func (w *processorWriter) Header() http.Header {
	return w.resp.Header()
}

func (w *processorWriter) WriteHeader(code int) {
	w.resp.WriteHeader(code)
}

func (w *processorWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}

type processorNode struct {
	name_        string
	findex       int
	requestType  reflect.Type
	responseType reflect.Type
}

func (n *processorNode) name() string {
	return n.name_
}

func (n *processorNode) handle(instance reflect.Value, ctx *context) {
	if ctx.compresser != nil {
		c, err := ctx.compresser.Writer(ctx.responseWriter)
		if err == nil {
			defer c.Close()
			ctx.responseWriter.Header().Set("Content-Encoding", ctx.compresser.Name())
			ctx.responseWriter = &processorWriter{
				resp:   ctx.responseWriter,
				writer: c,
			}
		}
	}

	// args := []reflect.Value{instance}
	var args []reflect.Value
	if n.requestType != nil {
		request := reflect.New(n.requestType)
		marshaller, ok := getMarshaller(ctx.requestMime)
		if !ok {
			http.Error(ctx.responseWriter, "can't find marshaller for"+ctx.mime, http.StatusBadRequest)
			return
		}
		err := marshaller.Unmarshal(ctx.request.Body, request.Interface())
		if err != nil {
			ctx.Error(http.StatusBadRequest, ctx.DetailError(-1, "marshal request to %s failed: %s", n.requestType.Name(), err))
			return
		}
		args = append(args, request.Elem())
	}

	ret := instance.Method(n.findex).Call(args)

	if ctx.isError || len(ret) == 0 {
		return
	}

	marshaller, ok := getMarshaller(ctx.mime)
	if !ok {
		http.Error(ctx.responseWriter, "can't find marshaller for"+ctx.mime, http.StatusBadRequest)
		return
	}
	err := marshaller.Marshal(ctx.responseWriter, ctx.name, ret[0].Interface())
	if err != nil {
		ctx.Error(http.StatusInternalServerError, ctx.DetailError(-1, "marshal response to %s failed: %s", ret[0].Type().Name(), err))
		return
	}
}

type streamingWriter struct {
	writer       io.Writer
	resp         http.ResponseWriter
	writedHeader bool
}

func (w *streamingWriter) Write(b []byte) (int, error) {
	if !w.writedHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.resp.Write(b)
}

func (w *streamingWriter) Header() http.Header {
	return w.resp.Header()
}

func (w *streamingWriter) WriteHeader(code int) {
	if w.writedHeader {
		return
	}
	w.writer.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", code, http.StatusText(code))))
	w.Header().Write(w.writer)
	w.writer.Write([]byte("\r\n"))
	w.writedHeader = true
}

type streamingNode struct {
	name_       string
	findex      int
	end         string
	requestType reflect.Type
}

func (n *streamingNode) name() string {
	return n.name_
}

func (n *streamingNode) handle(instance reflect.Value, ctx *context) {
	hj, ok := ctx.responseWriter.(http.Hijacker)
	if !ok {
		ctx.Error(http.StatusInternalServerError, ctx.DetailError(-1, "webserver doesn't support hijacking"))
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, ctx.DetailError(-1, "%s", err))
		return
	}
	defer conn.Close()

	resp := &processorWriter{
		resp:   ctx.responseWriter,
		writer: conn,
	}

	if ctx.compresser != nil {
		c, err := ctx.compresser.Writer(conn)
		if err == nil {
			defer c.Close()
			ctx.responseWriter.Header().Set("Content-Encoding", ctx.compresser.Name())
			resp.writer = c
		}
	}

	ctx.responseWriter = &streamingWriter{
		writer:       conn,
		resp:         resp,
		writedHeader: false,
	}

	stream, err := newStream(ctx, conn, n.end)
	if err != nil {
		ctx.Error(http.StatusBadRequest, ctx.DetailError(-1, "%s", err))
	}

	args := []reflect.Value{reflect.ValueOf(stream).Elem()}
	if n.requestType != nil {
		request := reflect.New(n.requestType)
		marshaller, ok := getMarshaller(ctx.requestMime)
		if !ok {
			http.Error(ctx.responseWriter, "can't find marshaller for"+ctx.mime, http.StatusBadRequest)
			return
		}
		err := marshaller.Unmarshal(ctx.request.Body, request.Interface())
		if err != nil {
			ctx.Error(http.StatusBadRequest, ctx.DetailError(-1, fmt.Sprintf("marshal request to %s failed: %s", n.requestType.Name(), err)))
			return
		}
		request = reflect.Indirect(request)
		args = append(args, request)
	}

	ctx.responseWriter.Header().Set("Connection", "keep-alive")
	instance.Method(n.findex).Call(args)
}
