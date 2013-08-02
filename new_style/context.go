package rest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

type Context interface {
	Path() string
	Query() url.Values
	RemoteAddr() string
	Header() http.Header
	Body() io.Reader

	Broadcast(room string, v interface{})

	ResponseHeader() http.Header

	marshaller() (Marshaller, error)
	responseCode(code StandardCode)
	response(v interface{})
	handleError(err error)
}

func rawPathToType(rawPath string) string {
	raw := []byte(rawPath)
	for {
		start := bytes.Index(raw, []byte(":"))
		if start < 0 {
			break
		}
		prefix := raw[:start]
		raw = raw[start+1:]
		end := bytes.Index(raw, []byte("/"))
		if end < 0 {
			raw = prefix
			break
		}
		raw = append(prefix, raw[end+1:]...)
	}
	start := bytes.Index(raw, []byte("*"))
	if start >= 0 {
		raw = raw[:start]
	}
	if l := len(raw); l > 1 && raw[l-1] == byte('/') {
		raw = raw[:l-1]
	}
	return string(raw)
}

var typeOfContext = reflect.TypeOf((*Context)(nil)).Elem()
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type StandardCode interface {
	Code() int
}

type StandardSuccess int

func (s StandardSuccess) Code() int { return int(s) }

const (
	OK                   StandardSuccess = StandardSuccess(http.StatusOK)
	Created                              = StandardSuccess(http.StatusCreated)
	Accepted                             = StandardSuccess(http.StatusAccepted)
	NonAuthoritativeInfo                 = StandardSuccess(http.StatusNonAuthoritativeInfo)
	NoContent                            = StandardSuccess(http.StatusNoContent)
	ResetContent                         = StandardSuccess(http.StatusResetContent)
	PartialContent                       = StandardSuccess(http.StatusPartialContent)
)

type StandardError struct {
	code int
	v    interface{}
	msg  string
}

func NewStandardError(code int, args ...interface{}) StandardError {
	ret := StandardError{
		code: code,
	}
	if len(args) > 0 {
		switch args[0].(type) {
		case string:
			ret.msg = fmt.Sprintf(args[0].(string), args[1:]...)
		default:
			ret.v = args[0]
		}
	}
	return ret
}

func (e StandardError) Code() int {
	return e.code
}

func (e StandardError) Error() string {
	if e.v != nil {
		return fmt.Sprintf("%+v", e.v)
	}
	return e.msg
}

func NotFound(args ...interface{}) StandardError {
	return NewStandardError(http.StatusNotFound, args...)
}

func BadRequest(args ...interface{}) StandardError {
	return NewStandardError(http.StatusBadRequest, args...)
}
