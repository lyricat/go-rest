/*
Package rest is a RESTful web-service framework. It make struct method to http.Hander automatically.

Define a service struct like this:

	type Conversation struct {
		Id   int
		To   string
		Text string
	}

	type RESTService struct {
		Service `prefix:"/prefix"`

		Hello    Processor `path:"/hello/(.*?)/to/(.*?)" method:"GET"`
		PostConv Processor `path:"/conversation/to/(.*?)" func:"PostConversation" method:"POST"`
		Conv     Processor `path:"/conversation/([0-9]+)" func:"GetConversation" method:"GET"`
		Watch    Streaming `path:"/conversation/streaming" method:"GET"`
	}

	// call /hello/{host}/to/{guest} and get a string.
	// parameters in url will pass to processor's function orderly.
	func (s RESTService) Hello_(host, guest string) string {
		return "hello from " + host + " to " + guest
	}

	// call /conversation/to/{people}, post a string as text and return a conversation object.
	// the post content will unmarshal to the last parameter of processor.
	// when save the conversation to db, send new conv to people through streaming api.
	func (s RESTService) PostConversation(to, post string) Conversation {
		conv := Conversation{
			Id:   1,
			To:   to,
			Text: post,
		}
		path, _ := s.Conv.Path(conv.Id)
		s.RedirectTo(path)
		s.Watch.Feed(to, conv)
		return conv
	}

	// call /conversation/{id} and get the conversation object.
	// rest will automatically convert id in url to int type. if convert failed, return bad request.
	func (s RESTService) GetConversation(id int) Conversation {
		return Conversation{
			Id:   1,
			To:   "to",
			Text: "post",
		}
	}

	// call /conversation/streaming?user=abc, create a long connection and get the conversation update of abc ASAP.
	// this function will be called when connecting to get a identity from request.
	// when feeding streaming, all connection with same identity will send the data.
	func (s RESTService) Watch_() string {
		return s.Request().URL.Query().Get("user")
	}

The field tag of RESTService configure the parameters of processor, like method, path, or function which 
will process the request.

The path of processor can capture arguments, which will pass to process function by order in path. Arguments
type can be string or int, or any type which kind is string or int. 

The default name of processor is the name of field postfix with "_", like Hello processor correspond Hello_ method.

Get the http.Handler from RESTService:

	handler, err := rest.New(new(RESTService))
	http.ListenAndServe("127.0.0.1:8080", handler)

Or use gorilla mux and work with other http handlers:

	// import "github.com/gorilla/mux"
	router := mux.NewRouter()
	handler, err := rest.New(new(RESTService))
	router.PathPrefix(handler.Prefix()).Handle(handler)
*/
package rest

import (
	"fmt"
	"net/http"
	"reflect"
)

// Rest handle the http request and call to correspond the handler(processor or streaming).
type Rest struct {
	prefix         string
	defaultCharset string
	defaultMime    string

	instance reflect.Value
	handlers []*node
}

// Create Rest instance from service instance
func New(i interface{}) (*Rest, error) {
	instance := reflect.ValueOf(i)
	if instance.Kind() != reflect.Struct && instance.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("%s's kind must struct or point to struct")
	}
	if instance.Kind() == reflect.Ptr {
		instance = instance.Elem()
	}
	t := instance.Type()
	serviceType, ok := t.FieldByName("Service")
	if !ok {
		return nil, fmt.Errorf("can't find restful.Service field")
	}
	if serviceType.Index[0] != 0 {
		return nil, fmt.Errorf("%s's 1st field must be restful.Service", t.Name())
	}

	serviceTag := serviceType.Tag
	service := instance.Field(0)
	prefix, mime, charset, err := initService(service, serviceTag)
	if err != nil {
		return nil, err
	}

	var handlers []*node
	for i, n := 0, instance.NumField(); i < n; i++ {
		handlerType := t.Field(i)
		if capital := handlerType.Name[0]; !('A' <= capital && capital <= 'Z') {
			continue
		}
		handler := instance.Field(i)
		if _, ok := handler.Interface().(nodeInterface); !ok {
			continue
		}

		node, err := newNode(t, prefix, handler, handlerType)
		if err != nil {
			return nil, err
		}
		handlers = append(handlers, node)
	}

	return &Rest{
		prefix:         prefix,
		defaultMime:    mime,
		defaultCharset: charset,
		handlers:       handlers,
		instance:       instance,
	}, nil
}

// Get the prefix of service.
func (s Rest) Prefix() string {
	return s.prefix
}

// Serve the http request.
func (s Rest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var errorCode int
	defer func() {
		r := recover()
		if r != nil {
			errorCode = http.StatusInternalServerError
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			http.Error(w, err.Error(), errorCode)
		}
	}()

	node := s.findNode(r)
	if node == nil {
		errorCode, err = http.StatusNotFound, fmt.Errorf("can't find node to process %s", r.URL.Path)
		return
	}

	args, e := node.match(r.Method, r.URL.Path)
	if e != nil {
		errorCode, err = http.StatusNotFound, e
		return
	}

	ctx, e := newContent(w, r, s.defaultMime, s.defaultCharset)
	if err != nil {
		errorCode, err = http.StatusBadRequest, e
		return
	}

	if req := node.request; req != nil {
		request := reflect.New(req)
		err = ctx.marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			errorCode, err = http.StatusBadRequest, fmt.Errorf("can't marshal request to type %s: %s", req, err)
			return
		}
		args = append(args, request.Elem())
	}

	service := s.instance.Field(0).Interface().(Service)
	service.ctx = ctx

	node.handle(s.instance, service.ctx, args)
}

func (s Rest) findNode(r *http.Request) *node {
	for _, h := range s.handlers {
		if h.method != r.Method {
			continue
		}
		if h.path.MatchString(r.URL.Path) {
			return h
		}
	}
	return nil
}
