package rest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

	w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=%s", ctx.mime, ctx.charset))

	if n.requestType != nil {
		request := reflect.New(n.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			marshaller.Marshal(w, marshaller.Error(-1, fmt.Sprintf("marshal request to %s failed: %s", n.requestType.Name(), err)))
			return
		}
		args = append(args, request.Elem())
	}
	ret := f.Call(args)

	if !ctx.isError && len(ret) > 0 {
		var writer io.Writer = w
		if ctx.compresser != nil {
			w.Header().Set("Content-Encoding", ctx.compresser.Name())
			var err error
			c, err := ctx.compresser.Writer(writer)
			defer c.Close()
			writer = c
			if err != nil {
				delete(w.Header(), "Content-Encoding")
				w.WriteHeader(http.StatusInternalServerError)
				marshaller.Marshal(w, marshaller.Error(-1, fmt.Sprintf("create compresser %s failed: %s", ctx.compresser.Name(), err)))
				return
			}
		}
		err := marshaller.Marshal(writer, ret[0].Interface())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			marshaller.Marshal(w, marshaller.Error(-1, fmt.Sprintf("marshal response to %s failed: %s", err)))
			return
		}
	}
}

type streamingWriter struct {
	conn         net.Conn
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

	hj, ok := w.(http.Hijacker)
	if !ok {
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", ctx.mime))
		w.WriteHeader(http.StatusInternalServerError)
		marshaller.Marshal(w, marshaller.Error(-2, "webserver doesn't support hijacking"))
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", ctx.mime))
		w.WriteHeader(http.StatusInternalServerError)
		marshaller.Marshal(w, marshaller.Error(-3, err.Error()))
		return
	}
	defer conn.Close()

	var writer io.Writer = conn
	if ctx.compresser != nil {
		c, err := ctx.compresser.Writer(writer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			marshaller.Marshal(w, marshaller.Error(-1, fmt.Sprintf("create compresser %s failed: %s", ctx.compresser.Name(), err)))
			return
		}
		defer c.Close()
		writer = c
	}

	ctx.responseWriter = &streamingWriter{
		conn:         conn,
		bufrw:        bufrw,
		header:       make(http.Header),
		compresser:   ctx.compresser,
		writer:       writer,
		writedHeader: false,
	}
	ctx.responseWriter.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", ctx.mime))

	stream := newStream(ctx, conn, n.end)

	var request reflect.Value
	if n.requestType != nil {
		request = reflect.New(n.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			marshaller.Marshal(w, marshaller.Error(-1, fmt.Sprintf("marshal request to %s failed: %s", n.requestType.Name(), err)))
			return
		}
		request = reflect.Indirect(request)
	}

	args := []reflect.Value{reflect.ValueOf(stream).Elem()}
	if n.requestType != nil {
		args = append(args, request)
	}

	ctx.responseWriter.Header().Set("Connection", "keep-alive")
	f.Call(args)
}
