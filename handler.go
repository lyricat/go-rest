package rest

import (
	"io"
	"net/http"
	"reflect"
	"strings"
)

// Handler is http process handler.
type Handler interface {
	// ServeHTTP serve the http request r with response writer w, vars is parameters grabbed from url path.
	ServeHTTP(w http.ResponseWriter, r *http.Request, vars map[string]string)
	Name() string
}

// Node is rest service node, it corresponde a handler.
type Node interface {
	// CreateHandler use service tag, field tag, fname and f function to create a handler with url path and method.
	CreateHandler(service reflect.StructTag, field reflect.StructTag, fname string, f reflect.Value) (path string, method string, handler Handler, err error)
}

func upperFirst(str string) string {
	return strings.ToUpper(str[:1]) + str[1:]
}

func parseHeaderField(req *http.Request, field string) (string, map[string]string) {
	splits := strings.Split(req.Header.Get(field), ";")
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

func getMarshallerFromRequest(mime string, marshaller Marshaller, r *http.Request) (string, Marshaller) {
	mime, ret := mime, marshaller
	if marshaller == nil {
		mime, ret = "application/json", jsonMarshaller
	}
	if mimeRequest, _ := parseHeaderField(r, "Content-Type"); mimeRequest != "" {
		if m, ok := getMarshaller(mimeRequest); ok {
			mime, ret = mimeRequest, m
		}
	}
	return mime, ret
}

func unmarshallFromReader(t reflect.Type, marshaller Marshaller, r io.Reader) (reflect.Value, error) {
	kind := t.Kind()
	if kind == reflect.Invalid {
		return reflect.ValueOf(nil), nil
	}
	var ret reflect.Value
	if kind == reflect.Ptr {
		ret = reflect.New(t.Elem())
	} else {
		ret = reflect.New(t)
	}
	err := marshaller.Unmarshal(r, ret.Interface())
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	if kind == reflect.Ptr {
		return ret, nil
	}
	return ret.Elem(), nil
}
