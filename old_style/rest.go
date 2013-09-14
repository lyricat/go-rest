/*
Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.

Define a service struct like this:

	type RestExample struct {
		rest.Service `prefix:"/prefix" mime:"application/json" charset:"utf-8"`

		CreateHello rest.Processor `method:"POST" path:"/hello"`
		GetHello    rest.Processor `method:"GET" path:"/hello/:to" func:"HandleHello"`
		Watch       rest.Streaming `method:"GET" path:"/hello/:to/streaming"`

		post  map[string]string
		watch map[string]chan string
	}

	type HelloArg struct {
		To   string `json:"to"`
		Post string `json:"post"`
	}

	// Post example:
	// > curl "http://127.0.0.1:8080/prefix/hello" -d '{"to":"rest", "post":"rest is powerful"}'
	//
	// No response
	func (r RestExample) HandleCreateHello(arg HelloArg) {
		r.post[arg.To] = arg.Post
		c, ok := r.watch[arg.To]
		if ok {
			select {
			case c <- arg.Post:
			default:
			}
		}
	}

	// Get example:
	// > curl "http://127.0.0.1:8080/prefix/hello/rest"
	//
	// Response:
	//   {"to":"rest","post":"rest is powerful"}
	func (r RestExample) HandleHello() HelloArg {
		to := r.Vars()["to"]
		post, ok := r.post[to]
		if !ok {
			r.Error(http.StatusNotFound, r.DetailError(2, "can't find hello to %s", to))
			return HelloArg{}
		}
		return HelloArg{
			To:   to,
			Post: post,
		}
	}

	// Streaming example:
	// > curl "http://127.0.0.1:8080/prefix/hello/rest/streaming"
	//
	// It create a long-live connection and will receive post content "rest is powerful"
	// when running post example.
	func (r RestExample) HandleWatch(s rest.Stream) {
		to := r.Vars()["to"]
		if to == "" {
			r.Error(http.StatusBadRequest, r.DetailError(3, "need to"))
			return
		}
		r.WriteHeader(http.StatusOK)
		c := make(chan string)
		r.watch[to] = c
		for {
			post := <-c
			s.SetDeadline(time.Now().Add(time.Second))
			err := s.Write(post)
			if err != nil {
				close(c)
				delete(r.watch, to)
				return
			}
		}
	}

The field tag of Service configure the parameters of processor, like method, path, or function which
will process the request.

The path of processor can capture arguments, which will pass to process function by order in path. Arguments
type can be string or int, or any type which kind is string or int.

The default name of handler is the name of field prefix with "Handle",
like Watch handelr correspond HandleWatch method.

Get the http.Handler from RestExample:

	handler, err := rest.New(&RestExample{
		post:  make(map[string]string),
		watch: make(map[string]chan string),
	})
	http.ListenAndServe("127.0.0.1:8080", handler)

Or use gorilla mux and work with other http handlers:

	// import "github.com/gorilla/mux"
	router := mux.NewRouter()
	handler, err := rest.New(&RestExample{
	    post:  make(map[string]string),
	    watch: make(map[string]chan string),
	})
	router.PathPrefix(handler.Prefix()).Handle(handler)
	http.ListenAndServe("127.0.0.1:8080", router)
*/
package rest

import (
	"fmt"
	"github.com/ant0ine/go-urlrouter"
	"net/http"
	"reflect"
)

// Rest handle the http request and call to correspond the handler(processor or streaming).
type Rest struct {
	instance       reflect.Value
	serviceIndex   int
	router         *urlrouter.Router
	prefix         string
	needCompress   bool
	defaultMime    string
	defaultCharset string
	ctxField       reflect.Value
}

// Create Rest instance from service instance
func New(s interface{}) (*Rest, error) {
	router := new(urlrouter.Router)

	instance := reflect.ValueOf(s)
	instance = reflect.Indirect(instance)
	t := instance.Type()
	serviceIndex, prefix, mime, charset := -1, "", "", ""
	needCompress := false
	for i, n := 0, instance.NumField(); i < n; i++ {
		field := instance.Field(i)
		if field.Type().String() == "rest.Service" {
			p, m, c, err := initService(field, t.Field(i).Tag)
			if err != nil {
				return nil, err
			}
			serviceIndex, prefix, mime, charset = i, p, m, c
			needCompress = t.Field(i).Tag.Get("compress") == "on"
		}
	}
	if serviceIndex < 0 {
		return nil, fmt.Errorf("%s doesn't contain rest.Service field.", t.Name())
	}
	for i, n := 0, instance.NumField(); i < n; i++ {
		node_ := instance.Field(i)
		field := t.Field(i)
		if !node_.CanAddr() {
			continue
		}
		if first := field.Name[0]; !('A' <= first && first <= 'Z') {
			continue
		}
		pNode, ok := node_.Addr().Interface().(node)
		if !ok {
			continue
		}

		method := field.Tag.Get("method")
		if method == "" {
			return nil, fmt.Errorf("%s node's tag must contain method", field.Name)
		}
		path := field.Tag.Get("path")

		formatter := pathToFormatter(prefix, path)
		handlers, paths, err := pNode.init(formatter, t, field.Name, field.Tag)
		if err != nil {
			return nil, err
		}
		for i := range handlers {
			router.Routes = append(router.Routes, urlrouter.Route{
				PathExp: fmt.Sprintf("/%s/%s", method, paths[i]),
				Dest:    handlers[i],
			})
		}
	}

	err := router.Start()
	if err != nil {
		return nil, err
	}

	return &Rest{
		instance:       instance,
		serviceIndex:   serviceIndex,
		router:         router,
		prefix:         prefix,
		needCompress:   needCompress,
		defaultMime:    mime,
		defaultCharset: charset,
		ctxField:       instance.Field(serviceIndex).FieldByName("context"),
	}, nil
}

// Get the url prefix of service.
func (r *Rest) Prefix() string {
	return r.prefix
}

// Serve the http request.
func (re *Rest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if method := r.URL.Query().Get("_method"); method != "" {
		r.Method = method
	}
	r.URL.Path = fmt.Sprintf("/%s/%s", r.Method, path)
	dest, vars := re.router.FindRouteFromURL(r.URL)
	if dest == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	r.URL.Path = path

	handler := dest.Dest.(handler)

	if !re.needCompress {
		delete(r.Header, "Accept-Encoding")
	}

	ctx, err := newContext(w, r, vars, re.defaultMime, re.defaultCharset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx.name = handler.name()

	ctx.responseWriter.Header().Set("Content-Type", fmt.Sprintf("%s; charset=%s", ctx.mime, ctx.charset))

	re.ctxField.Set(reflect.ValueOf(ctx))

	handler.handle(re.instance, ctx)
}
