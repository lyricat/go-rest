package rest

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ant0ine/go-urlrouter"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPTransport struct {
	DefaultMime           string
	DefaultCharset        string
	StreamingNotification bool
	MaxConnectionPerRoom  int
	StreamingPingInterval time.Duration

	router     *urlrouter.Router
	roomSetter map[string]RoomSetter
}

func (t *HTTPTransport) Add(prefix string, handler *Handler) error {
	prefix = normalizePath(prefix)
	routes := make([]urlrouter.Route, len(handler.routes))
	for i, route := range handler.routes {
		route.PathExp = prefix + route.PathExp
		node := route.Dest.(*node)
		var methods []string
		for k := range node.handlers {
			methods = append(methods, k)
			node.handlers[k].type_ = rawPathToType(route.PathExp)
		}
		node.methods = strings.Join(methods, ", ")
		routes[i] = route
	}
	if t.router == nil {
		t.router = new(urlrouter.Router)
	}
	t.router.Routes = append(t.router.Routes, routes...)
	return t.router.Start()
}

func (t *HTTPTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if m, ok := r.URL.Query()["_method"]; ok {
		method = m[0]
	}

	if r.URL.Path == "/" && method == "WATCH" {
		t.Streaming(w, r)
		return
	}

	route, vars := t.router.FindRouteFromURL(r.URL)
	if route == nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	node := route.Dest.(*node)

	handler, ok := node.handlers[method]
	if !ok {
		w.Header().Set("Allow", node.methods)
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	delete(query, "_method")

	ctx := &HTTPContext{
		r:           r,
		w:           w,
		node:        handler,
		defaultMime: t.DefaultMime,
		query:       mergeQuery(query, vars),
	}
	handler.handler.handle(ctx)
}

func (t *HTTPTransport) Streaming(w http.ResponseWriter, r *http.Request) {
	ctx := &HTTPContext{
		r:           r,
		w:           w,
		defaultMime: t.DefaultMime,
	}
	err := ctx.parseResponseMarshaller()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mime, marshaler := ctx.respMime, ctx.respMarshaller

	err = r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	recv := make(chan interface{})
	var okRooms []string
	for _, room := range r.Form["room"] {
		err := roomListen(room, t.MaxConnectionPerRoom, recv)
		if err != nil {
			continue
		}
		okRooms = append(okRooms, room)
	}

	if len(okRooms) == 0 {
		http.Error(w, "no enough connection slot", http.StatusBadRequest)
		return
	}

	defer func() {
		for _, room := range okRooms {
			roomUnlisten(room, recv)
		}
	}()

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

	_, err = bufrw.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", http.StatusOK, http.StatusText(http.StatusOK)))
	if err != nil {
		return
	}
	header := make(http.Header)
	header.Set("Content-Type", mime)
	header.Set("Connection", "keep-alive")
	err = header.Write(bufrw)
	if err != nil {
		return
	}
	_, err = bufrw.WriteString("\r\n")
	if err != nil {
		return
	}

	for _, init := range r.Form["init"] {
		v, err := t.streamingGet(init, r)
		if err != nil {
			continue
		}
		marshaler.Marshal(bufrw, r.URL.Path, v)
	}

	err = bufrw.Flush()
	if err != nil {
		return
	}

	for {
		buf := make([]byte, 1)
		select {
		case v := <-recv:
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			err = marshaler.Marshal(bufrw, "", v)
			if err != nil {
				return
			}
			err = bufrw.Flush()
			if err != nil {
				return
			}
		case <-time.After(t.StreamingPingInterval):
			conn.SetReadDeadline(time.Now().Add(time.Second / 10))
			_, err = conn.Read(buf)
			if e, ok := err.(net.Error); err == nil || (ok && e.Timeout()) {
				continue
			}
			return
		}
	}
}

func (t *HTTPTransport) streamingGet(url string, r *http.Request) (interface{}, error) {
	ques := strings.Index(url, "?")
	if ques >= 0 {
		r.URL.Path = url[:ques]
		r.URL.RawQuery = url[ques+1:]
	} else {
		r.URL.Path = url
		r.URL.RawQuery = ""
	}

	route, vars := t.router.FindRouteFromURL(r.URL)
	if route == nil {
		return nil, fmt.Errorf("response: 404")
	}
	node := route.Dest.(*node)

	handler, ok := node.handlers["GET"]
	if !ok {
		return nil, fmt.Errorf("response: 404")
	}

	ctx := &HTTPStreamingContext{
		r:      r,
		type_:  handler.type_,
		query:  mergeQuery(r.URL.Query(), vars),
		buf:    bytes.NewBuffer(nil),
		header: make(http.Header),
		code:   http.StatusOK,
	}

	handler.handler.handle(ctx)
	if ctx.code != http.StatusOK {
		return nil, fmt.Errorf("response: %d", ctx.code)
	}

	return ctx.v, nil
}

func mergeQuery(query url.Values, vars map[string]string) url.Values {
	for k, v := range vars {
		if q, ok := query[k]; ok {
			query[k] = append(q, v)
		} else {
			query[k] = []string{v}
		}
	}
	return query
}

func normalizePath(prefix string) string {
	if prefix == "" {
		prefix = "/"
	} else {
		if prefix[0] != '/' {
			prefix = "/" + prefix
		}
		if l := len(prefix); l > 1 && prefix[l-1] == '/' {
			prefix = prefix[:l-1]
		}
	}
	return prefix
}

func parseHeaderField(r *http.Request, field string) (string, map[string]string) {
	splits := strings.Split(r.Header.Get(field), ";")
	ret := strings.Trim(splits[0], " ")
	splits = splits[1:]
	var pair map[string]string
	if len(splits) > 0 {
		pair = make(map[string]string)
		for _, s := range splits {
			s = strings.Trim(s, " ")
			if s == "" {
				continue
			}
			i := strings.Index(s, "=")
			if i > 0 {
				pair[s[:i]] = s[i+1:]
			} else {
				pair[s] = ""
			}
		}
	}
	return ret, pair
}

type HTTPContext struct {
	r           *http.Request
	w           http.ResponseWriter
	node        *namedNode
	defaultMime string
	query       url.Values

	reqMime        string
	reqMarshaller  Marshaller
	respMime       string
	respMarshaller Marshaller
}

func (c *HTTPContext) Path() string {
	return c.r.URL.Path
}

func (c *HTTPContext) Query() url.Values {
	return c.query
}

func (c *HTTPContext) RemoteAddr() string {
	return c.r.RemoteAddr
}

func (c *HTTPContext) Header() http.Header {
	return c.r.Header
}

func (c *HTTPContext) Body() io.Reader {
	return c.r.Body
}

func (c *HTTPContext) Broadcast(room string, v interface{}) {
	roomSend(room, v)
}

func (c *HTTPContext) ResponseHeader() http.Header {
	return c.w.Header()
}

func (c *HTTPContext) marshaller() (Marshaller, error) {
	err := c.parseRequestMarshaller()
	if err != nil {
		return nil, err
	}
	return c.reqMarshaller, nil
}

func (c *HTTPContext) responseCode(code StandardCode) {
	c.w.WriteHeader(code.Code())
}

func (c *HTTPContext) response(v interface{}) {
	if c.parseResponseMarshaller() != nil {
		return
	}
	c.respMarshaller.Marshal(c.w, c.node.name, v)
}

func (c *HTTPContext) handleError(err error) {
	if stdErr, ok := err.(StandardError); ok {
		if stdErr.v != nil {
			if c.parseResponseMarshaller() != nil {
				return
			}
			c.w.WriteHeader(stdErr.code)
			c.respMarshaller.Marshal(c.w, c.node.name, stdErr.v)
			return
		}
		http.Error(c.w, stdErr.msg, stdErr.code)
		return
	}
	http.Error(c.w, err.Error(), http.StatusInternalServerError)
}

func (c *HTTPContext) parseRequestMarshaller() error {
	if c.reqMarshaller != nil {
		return nil
	}
	mime, _ := parseHeaderField(c.r, "Content-Type")
	marshaller, ok := getMarshaller(mime)
	if ok {
		c.reqMime, c.reqMarshaller = mime, marshaller
		return nil
	}
	mime = c.defaultMime
	marshaller, ok = getMarshaller(mime)
	if ok {
		c.reqMime, c.reqMarshaller = mime, marshaller
		return nil
	}
	http.Error(c.w, "can't unmarshal request", http.StatusBadRequest)
	return errors.New("can't unmarshal request")
}

func (c *HTTPContext) parseResponseMarshaller() error {
	if c.respMarshaller != nil {
		return nil
	}
	mime := c.r.Header.Get("Accept")
	marshaller, ok := getMarshaller(mime)
	if ok {
		c.respMime, c.respMarshaller = mime, marshaller
		return nil
	}
	mime, _ = parseHeaderField(c.r, "Content-Type")
	marshaller, ok = getMarshaller(mime)
	if ok {
		c.respMime, c.respMarshaller = mime, marshaller
		return nil
	}
	mime = c.defaultMime
	marshaller, ok = getMarshaller(mime)
	if ok {
		c.respMime, c.respMarshaller = mime, marshaller
		return nil
	}
	http.Error(c.w, "can't marshal response", http.StatusBadRequest)
	return errors.New("can't marshal response")
}

type HTTPStreamingContext struct {
	r      *http.Request
	type_  string
	query  url.Values
	v      interface{}
	buf    *bytes.Buffer
	header http.Header
	code   int
}

func (c *HTTPStreamingContext) Path() string {
	return c.r.URL.Path
}

func (c *HTTPStreamingContext) Query() url.Values {
	return c.query
}

func (c *HTTPStreamingContext) RemoteAddr() string {
	return c.r.RemoteAddr
}

func (c *HTTPStreamingContext) Header() http.Header {
	return c.r.Header
}

func (c *HTTPStreamingContext) Body() io.Reader {
	return c.buf
}

func (c *HTTPStreamingContext) Broadcast(room string, v interface{}) {
	roomSend(room, v)
}

func (c *HTTPStreamingContext) ResponseHeader() http.Header {
	return c.header
}

func (c *HTTPStreamingContext) marshaller() (Marshaller, error) {
	return nil, nil
}

func (c *HTTPStreamingContext) responseCode(code StandardCode) {
	c.code = code.Code()
}

func (c *HTTPStreamingContext) response(v interface{}) {
	c.v = v
}

func (c *HTTPStreamingContext) handleError(err error) {
	if stdErr, ok := err.(StandardError); ok {
		c.code = stdErr.code
		return
	}
	c.code = http.StatusInternalServerError
}
