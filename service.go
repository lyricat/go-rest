package rest

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

/*
Define the service.

Valid tag:

 - root: The root path of http request. All processor's path will prefix with root path.

The priority of value is: value in Service, value in tag, default.

To be implement:
 - mime: Define the default mime of all processor in this service.
 - charset: Define the default charset of all processor in this service.
*/
type Service struct {
	*innerService

	// Set the service root path, it will over right root in tag.
	Root string

	// Set the service default mime, it will over right mime in tag.
	DefaultMime string

	// Set the service default charset, it will over right charset in tag.
	DefaultCharset string
}

// Return the http request instance.
func (s Service) Request() *http.Request {
	return s.ctx.request
}

// Header returns the header map that will be sent.
func (s Service) Response(code int) {
	s.ctx.status = code
}

// Get the response header.
func (s Service) Header() http.Header {
	return s.ctx.header
}

// Error replies to the request with the specified error message and HTTP code.
func (s Service) Error(code int, err error) {
	s.ctx.status = code
	s.ctx.error = err
}

// Redirect to the specified path.
func (s Service) RedirectTo(path string) {
	s.ctx.status = http.StatusTemporaryRedirect
	s.Header().Set("Location", path)
}

func initService(service reflect.Value, tag reflect.StructTag) error {
	mime := service.FieldByName("DefaultMime").Interface().(string)
	if mime == "" {
		mime = tag.Get("mime")
	}
	if mime == "" {
		mime = "application/json"
	}

	charset := service.FieldByName("DefaultCharset").Interface().(string)
	if charset == "" {
		charset = tag.Get("charset")
	}
	if charset == "" {
		charset = "utf-8"
	}

	root := service.FieldByName("Root").Interface().(string)
	if root == "" {
		root = tag.Get("root")
	}
	if root == "" {
		root = "/"
	}

	service.Field(0).Set(reflect.ValueOf(&innerService{
		root:           root,
		defaultMime:    mime,
		defaultCharset: charset,
		tag:            tag,
	}))
	return nil
}

type context struct {
	request *http.Request
	status  int
	header  http.Header
	error   error
}

type innerService struct {
	root           string
	defaultCharset string
	defaultMime    string
	tag            reflect.StructTag

	instance   interface{}
	processors []Processor
	ctx        *context
}

func (s innerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	val := reflect.ValueOf(s.instance)
	inner := val.Field(0).Field(0).Interface().(*innerService)
	inner.ctx = &context{r, http.StatusOK, w.Header(), nil}

	f := val.Method(handler.funcIndex)
	resp := f.Call(args)

	w.WriteHeader(inner.ctx.status)
	if 200 <= inner.ctx.status && inner.ctx.status <= 399 && len(resp) > 0 {
		marshaller.Marshal(w, resp[0].Interface())
	} else {
		w.Write([]byte(inner.ctx.error.Error()))
	}

}

func (s innerService) findProcessor(r *http.Request) (Processor, bool) {
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

func (s innerService) getContentTypeFromRequset(r *http.Request) (string, string) {
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

func parseRealm(tag reflect.StructTag) []string {
	return strings.Split(tag.Get("realm"), ",")
}
