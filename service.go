package rest

import (
	"net/http"
	"reflect"
)

/*
Define the service.

Valid tag:

 - prefix: The prefix path of http request. All processor's path will prefix with prefix path.

The priority of value is: value in Service, value in tag, default.

To be implement:
 - mime: Define the default mime of all processor in this service.
 - charset: Define the default charset of all processor in this service.
*/
type Service struct {
	*innerService

	// Set the service prefix path, it will over right prefix in tag.
	Prefix string

	// Set the service default mime, it will over right mime in tag.
	DefaultMime string

	// Set the service default charset, it will over right charset in tag.
	DefaultCharset string
}

// Return the http request instance.
func (s Service) Request() *http.Request {
	return s.ctx.request
}

// Variables from url.
func (s Service) Vars() map[string]string {
	return s.ctx.vars
}

// Write response code and header. Same as http.ResponseWriter.WriteHeader(int)
func (s Service) WriteHeader(code int) {
	if s.ctx.headerWriter != nil {
		s.ctx.headerWriter.writeHeader(code)
	} else {
		s.ctx.responseWriter.WriteHeader(code)
	}
}

// Get the response header.
func (s Service) Header() http.Header {
	return s.ctx.responseWriter.Header()
}

// Error replies to the request with the specified error message and HTTP code.
func (s Service) Error(code int, err error) {
	http.Error(s.ctx.responseWriter, err.Error(), code)
	s.ctx.isError = true
}

// Redirect to the specified path.
func (s Service) RedirectTo(path string) {
	s.Header().Set("Location", path)
	s.WriteHeader(http.StatusTemporaryRedirect)
}

func initService(service reflect.Value, tag reflect.StructTag) (string, string, string, error) {
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

	prefix := service.FieldByName("Prefix").Interface().(string)
	if prefix == "" {
		prefix = tag.Get("prefix")
	}
	if prefix == "" {
		prefix = "/"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if l := len(prefix); l > 2 && prefix[l-1] == '/' {
		prefix = prefix[:l-1]
	}

	service.Set(reflect.ValueOf(Service{
		innerService:   new(innerService),
		Prefix:         prefix,
		DefaultMime:    mime,
		DefaultCharset: charset,
	}))
	return prefix, mime, charset, nil
}

type innerService struct {
	ctx *context
}
