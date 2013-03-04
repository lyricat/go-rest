/*
Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.

Define a service struct like this:

	type RESTService struct {
		Service `root:"/root"`

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
*/
package rest

import (
	"fmt"
	"net/http"
	"reflect"
)

// Create http.Handler from service instance
func New(i interface{}) (http.Handler, error) {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Struct && v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("%s's kind must struct or point to struct")
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	serviceType, ok := t.FieldByName("Service")
	if !ok {
		return nil, fmt.Errorf("can't find restful.Service field")
	}
	if serviceType.Index[0] != 0 {
		return nil, fmt.Errorf("%s's 1st field must be restful.Service", t.Name())
	}

	serviceTag := serviceType.Tag
	service := v.Field(0)
	err := initService(service, serviceTag)
	if err != nil {
		return nil, err
	}
	inner := service.Field(0).Interface().(*innerService)
	root := inner.root

	var processors []Processor
	for i, n := 0, v.NumField(); i < n; i++ {
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

		handler := v.Field(i)
		err := initProcessor(root, handler, handlerType.Tag, f)
		if err != nil {
			return nil, fmt.Errorf("%s %s", handlerType.Name, err)
		}

		processors = append(processors, handler.Interface().(Processor))
	}

	inner.processors = processors
	inner.instance = v.Interface()
	return inner, nil
}
