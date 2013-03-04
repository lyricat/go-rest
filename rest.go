/*
Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.

Define a service struct like this:

	type RESTService struct {
		Service `prefix:"/prefix"`

		Hello    Processor `path:"/hello/(.*?)/to/(.*?)" method:"GET"`
		PostConv Processor `path:"/conversation" func:"PostConversation" method:"POST"`
		Conv     Processor `path:"/conversation/([0-9]+)" func:"GetConversation" method:"GET"`
	}

	func (s RESTService) Hello_(host, guest string) string {
		return "hello from " + host + " to " + guest
	}

	func (s RESTService) PostConversation(post string) string {
		path, _ := s.Conv.Path(1)
		s.RedirectTo(path)
		return "just post: " + post
	}

	func (s RESTService) GetConversation(id int) string {
		return fmt.Sprintf("get post id %d", id)
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
	"strings"
)

type context struct {
	request *http.Request
	status  int
	header  http.Header
	error   error
}

// Rest handle the http request and call to correspond the processor.
type Rest struct {
	prefix         string
	defaultCharset string
	defaultMime    string

	instance   reflect.Value
	processors []Processor
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

	var processors []Processor
	for i, n := 0, instance.NumField(); i < n; i++ {
		handlerType := t.Field(i)
		if handlerType.Type.Name() != "Processor" {
			continue
		}

		funcName := handlerType.Tag.Get("func")
		if funcName == "" {
			funcName = handlerType.Name + "_"
		}
		f, ok := t.MethodByName(funcName)
		if !ok {
			return nil, fmt.Errorf("%s can't find method with name %s", t.Name(), funcName)
		}

		handler := instance.Field(i)
		err := initProcessor(prefix, handler, handlerType.Tag, f)
		if err != nil {
			return nil, fmt.Errorf("%s %s", handlerType.Name, err)
		}

		processors = append(processors, handler.Interface().(Processor))
	}

	return &Rest{
		prefix:         prefix,
		defaultMime:    mime,
		defaultCharset: charset,
		processors:     processors,
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

	handler, ok := s.findProcessor(r)
	if !ok {
		errorCode, err = http.StatusNotFound, fmt.Errorf("can't find handler to process %s", r.URL.Path)
		return
	}

	mime, _ := s.getContentTypeFromRequset(r)
	marshaller, ok := getMarshaller(mime)
	if !ok {
		mime = s.defaultMime
		marshaller, ok = getMarshaller(mime)
	}
	if !ok {
		errorCode, err = http.StatusBadRequest, fmt.Errorf("can't find %s marshaller", mime)
		return
	}

	args, argErr := handler.getArgs(r.URL.Path)
	if err != nil {
		errorCode, err = http.StatusNotFound, argErr
		return
	}

	if handler.requestType != nil {
		request := reflect.New(handler.requestType)
		err = marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			errorCode, err = http.StatusBadRequest, fmt.Errorf("can't marshal request to type %s: %s", handler.requestType, err)
			return
		}
		args = append(args, request.Elem())
	}

	inner := s.instance.Field(0).Field(0).Interface().(*innerService)
	inner.ctx = &context{r, http.StatusOK, w.Header(), nil}

	f := s.instance.Method(handler.funcIndex)
	resp := f.Call(args)

	w.WriteHeader(inner.ctx.status)
	if 200 <= inner.ctx.status && inner.ctx.status <= 399 && len(resp) > 0 {
		marshaller.Marshal(w, resp[0].Interface())
	} else if inner.ctx.error != nil {
		w.Write([]byte(inner.ctx.error.Error()))
	}

}

func (s Rest) findProcessor(r *http.Request) (Processor, bool) {
	for _, h := range s.processors {
		if h.method != r.Method {
			continue
		}
		if h.path.MatchString(r.URL.Path) {
			return h, true
		}
	}
	return Processor{}, false
}

func (s Rest) getContentTypeFromRequset(r *http.Request) (string, string) {
	contentType := strings.Split(r.Header.Get("Content-Type"), ";")
	mime, charset := "", ""
	if len(contentType) > 0 {
		mime = strings.Trim(contentType[0], " \t")
	}
	if len(contentType) > 1 {
		for _, property := range contentType[1:] {
			property = strings.Trim(property, " \t")
			if len(property) > 8 && property[:8] == "charset=" {
				charset = property[8:]
				break
			}
		}
	}
	if mime == "" {
		mime = s.defaultMime
	}
	if charset == "" {
		charset = s.defaultCharset
	}

	return mime, charset
}
