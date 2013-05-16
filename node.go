package rest

import (
	"bufio"
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
	funcIndex    int
	requestType  reflect.Type
	responseType reflect.Type
}

func (n *processorNode) handle(instance reflect.Value, ctx *context) {
	r := ctx.request
	w := ctx.responseWriter
	marshaller := ctx.marshaller
	f := instance.Method(n.funcIndex)
	args := make([]reflect.Value, 1, 2)
	args[0] = reflect.ValueOf(ctx)

	w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=%s", ctx.mime, ctx.charset))

	if ctx.compresser != nil {
		c, err := ctx.compresser.Writer(ctx.responseWriter)
		if err == nil {
			defer c.Close()
			ctx.responseWriter = &processorWriter{
				resp:   ctx.responseWriter,
				writer: c,
			}
			w.Header().Set("Content-Encoding", ctx.compresser.Name())
		}
	}

	if n.requestType != nil {
		request := reflect.New(n.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			ctx.Error(http.StatusBadRequest, -1, "marshal request to %s failed: %s", n.requestType.Name(), err)
			return
		}
		args = append(args, request.Elem())
	}
	ret := f.Call(args)

	if !ctx.isError && len(ret) > 0 {
		err := marshaller.Marshal(ctx.responseWriter, ret[0].Interface())
		if err != nil {
			ctx.Error(http.StatusInternalServerError, -1, "marshal response to %s failed: %s", err)
			return
		}
	}
}

type streamingWriter struct {
	bufrw        *bufio.ReadWriter
	header       http.Header
	compresser   Compresser
	writer       io.Writer
	writedHeader bool
}

func (w *streamingWriter) Write(b []byte) (int, error) {
	if !w.writedHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(b)
}

func (w *streamingWriter) Header() http.Header {
	return w.header
}

func (w *streamingWriter) WriteHeader(code int) {
	if w.writedHeader {
		return
	}
	if w.compresser != nil {
		w.header.Set("Content-Encoding", w.compresser.Name())
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

	w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", ctx.mime))

	var request reflect.Value
	if n.requestType != nil {
		request = reflect.New(n.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			ctx.Error(http.StatusBadRequest, -1, "marshal request to %s failed: %s", n.requestType.Name(), err)
			return
		}
		request = reflect.Indirect(request)
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		ctx.Error(http.StatusInternalServerError, -1, "webserver doesn't support hijacking")
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, -1, "%s", err)
		return
	}
	defer conn.Close()

	var writer io.Writer = conn
	if ctx.compresser != nil {
		c, err := ctx.compresser.Writer(writer)
		if err != nil {
			ctx.Error(http.StatusBadRequest, -1, "create compresser %s failed: %s", ctx.compresser.Name(), err)
			return
		}
		defer c.Close()
		writer = c
		w.Header().Set("Content-Encoding", ctx.compresser.Name())
	}

	ctx.responseWriter = &streamingWriter{
		bufrw:        bufrw,
		header:       make(http.Header),
		compresser:   ctx.compresser,
		writer:       writer,
		writedHeader: false,
	}
	for k, v := range w.Header() {
		ctx.responseWriter.Header()[k] = v
	}
	ctx.responseWriter.Header().Set("Connection", "keep-alive")

	stream := newStream(ctx, conn, n.end)

	args := []reflect.Value{reflect.ValueOf(stream).Elem()}
	if n.requestType != nil {
		args = append(args, request)
	}

	f.Call(args)
}
