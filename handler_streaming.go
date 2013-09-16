package rest

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"time"
)

// StreamContext is informations when streaming.
type StreamContext interface {
	// Context is normal http context.
	Context

	// SetWriteDeadline sets the deadline for future Write calls.
	SetWriteDeadline(t time.Time) error

	// Ping check the streaming connection is still alive.
	Ping() error
}

// Streaming is node using for streaming request. It's tag has below parameters :
//  - method: http request method which need be handled.
//  - route: service's tag prefix add route is http request url path.
//  - path: path will ignore service's prefix tag, and use as url path.
type Streaming struct{}

// CreateHandler create streaming handler.
func (p Streaming) CreateHandler(serviceTag reflect.StructTag, fieldTag reflect.StructTag, fname string, f reflect.Value) (string, string, Handler, error) {
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

	endline := fieldTag.Get("end")

	t := f.Type()
	if t.NumIn() != 1 && t.NumIn() != 2 {
		return "", "", nil, fmt.Errorf("handler method %s should have 1 or 2 input parameters", fname)
	}
	if t.NumOut() != 0 {
		return "", "", nil, fmt.Errorf("handler method %s should not have return parameter", fname)
	}
	p0 := t.In(0)
	if p0.Kind() != reflect.Interface || p0.Name() != "StreamContext" {
		return "", "", nil, fmt.Errorf("handler method %s's 1st parameter must be rest.Context", fname)
	}
	var p1 reflect.Type
	if t.NumIn() == 2 {
		p1 = t.In(1)
	}

	return path, method, &streamHandler{fname, endline, mime, marshaller, p1, f}, nil
}

type streamHandler struct {
	name       string
	endline    string
	mime       string
	marshaller Marshaller
	inputType  reflect.Type
	f          reflect.Value
}

func (h *streamHandler) Name() string {
	return h.name
}

func (h *streamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	mime, marshaller := getMarshallerFromRequest(h.mime, h.marshaller, r)

	ctx, err := newStreamContext(h.name, marshaller, "utf-8", vars, h.endline, r, w)
	if err != nil {
		ctx := newBaseContext(h.name, marshaller, "utf-8", vars, r, w)
		ctx.Return(http.StatusInternalServerError, "%s", err)
		return
	}
	defer ctx.close()
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

type streamResponseWriter struct {
	header         http.Header
	hasWriteHeader bool
	writer         io.Writer
}

func newStreamResponseWriter(writer io.Writer) *streamResponseWriter {
	return &streamResponseWriter{
		header:         nil,
		hasWriteHeader: false,
		writer:         writer,
	}
}

func (w *streamResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *streamResponseWriter) WriteHeader(code int) {
	if w.hasWriteHeader {
		return
	}
	w.writer.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", code, http.StatusText(code))))
	if w.header != nil {
		w.header.Write(w.writer)
	}
	w.writer.Write([]byte("\r\n"))
	w.hasWriteHeader = true
}

func (w *streamResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.writer.Write(p)
}

type streamContext struct {
	*baseContext

	endLine string
	conn    net.Conn
	bufrw   *bufio.ReadWriter
}

func newStreamContext(handlerName string, marshaller Marshaller, charset string, vars map[string]string, endLine string, req *http.Request, resp http.ResponseWriter) (*streamContext, error) {
	hj, ok := resp.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("webserver doesn't support hijacking")
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	resp = newStreamResponseWriter(bufrw)
	baseContext := newBaseContext(handlerName, marshaller, charset, vars, req, resp)
	return &streamContext{
		baseContext: baseContext,
		endLine:     endLine,
		conn:        conn,
		bufrw:       bufrw,
	}, nil
}

func (ctx *streamContext) Return(code int, fmtAndArgs ...interface{}) {
	ctx.baseContext.Return(code, fmtAndArgs...)
	ctx.bufrw.Flush()
}

func (ctx *streamContext) Render(v interface{}) error {
	if err := ctx.baseContext.Render(v); err != nil {
		return err
	}
	if len(ctx.endLine) > 0 {
		if _, err := ctx.bufrw.Write([]byte(ctx.endLine)); err != nil {
			return err
		}
	}
	if err := ctx.bufrw.Flush(); err != nil {
		return err
	}
	return nil
}

func (ctx *streamContext) SetWriteDeadline(t time.Time) error {
	return ctx.conn.SetWriteDeadline(t)
}

func (ctx *streamContext) Ping() error {
	ctx.conn.SetReadDeadline(time.Now().Add(time.Second / 100))
	p := make([]byte, 1)
	_, err := ctx.conn.Read(p)
	if err == nil {
		return nil
	}
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return nil
	}
	return err
}

func (ctx *streamContext) close() {
	ctx.Response().WriteHeader(http.StatusOK)
	ctx.bufrw.Flush()
	ctx.conn.Close()
}
