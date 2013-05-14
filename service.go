package rest

import (
	"net/http"
	"reflect"
)

/*
Define the service.

Valid tag:

 - prefix: The prefix path of http request. All processor's path will prefix with prefix path.
 - mime: Define the default mime of all processor in this service.
 - compress: If value is "on", it will compress response using "Accept-Encoding" in request header.

The priority of value is: value in Service, value in tag, default.

To be implement:
 - charset: Define the default charset of all processor in this service.
*/
type Service struct {
	// Set the service prefix path, it will over right prefix in tag.
	Prefix string

	// Set the service default mime, it will over right mime in tag.
	DefaultMime string

	// Set the service default charset, it will over right charset in tag.
	DefaultCharset string

	ctx *context
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
	s.ctx.responseWriter.WriteHeader(code)
}

// Get the response header.
func (s Service) Header() http.Header {
	return s.ctx.responseWriter.Header()
}

// Get Default format error, which is like:
//
//     type Error struct {
//         Code    int
//         Message string
//     }
//
// And it will marshal to special mime-type when calling with Service.Error.
func (s Service) GetError(code int, message string) error {
	return s.ctx.marshaller.Error(code, message)
}

// Error replies to the request with the specified error message and HTTP code.
func (s Service) Error(code int, err error) {
	s.WriteHeader(code)
	s.ctx.marshaller.Marshal(s.ctx.responseWriter, err)
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
		Prefix:         prefix,
		DefaultMime:    mime,
		DefaultCharset: charset,
	}))
	return prefix, mime, charset, nil
}
