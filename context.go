package rest

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type headerWriter interface {
	writeHeader(int)
}

type context struct {
	name           string
	request        *http.Request
	vars           map[string]string
	requestMime    string
	requestCharset string
	responseWriter http.ResponseWriter
	mime           string
	charset        string
	compresser     Compresser
	isError        bool
}

func newContext(w http.ResponseWriter, r *http.Request, vars map[string]string, defaultMime, defaultCharset string) (*context, error) {
	requestMime, v := parseHeaderField(r, "Content-Type")
	if requestMime == "" {
		requestMime = defaultMime
	}
	if _, ok := getMarshaller(requestMime); !ok {
		requestMime = defaultMime
	}
	if _, ok := getMarshaller(requestMime); !ok {
		return nil, errors.New("can't find marshaller for " + requestMime)
	}
	requestCharset := v["charset"]
	if requestCharset == "" {
		requestCharset = defaultCharset
	}
	mime := r.Header.Get("Accept")
	if mime == "" {
		mime = requestMime
	}
	if _, ok := getMarshaller(mime); !ok {
		mime = defaultMime
	}
	if _, ok := getMarshaller(mime); !ok {
		return nil, errors.New("can't find marshaller for " + mime)
	}
	charset := r.Header.Get("Accept-Charset")
	if charset == "" {
		charset = requestCharset
	}

	encoding := r.Header.Get("Accept-Encoding")
	var compresser Compresser
	if encoding != "" {
		for _, name := range strings.Split(encoding, ",") {
			name = strings.Trim(name, " ")
			if c, ok := getCompresser(name); ok {
				compresser = c
				break
			}
		}
	}

	return &context{
		request:        r,
		vars:           vars,
		requestMime:    requestMime,
		requestCharset: requestCharset,
		mime:           mime,
		charset:        charset,
		compresser:     compresser,
		responseWriter: w,
		isError:        false,
	}, nil
}

// Return the http request instance.
func (c *context) Request() *http.Request {
	return c.request
}

// Variables from url.
func (c *context) Vars() map[string]string {
	return c.vars
}

// Write response code and header. Same as http.ResponseWriter.WriteHeader(int)
func (c *context) WriteHeader(code int) {
	c.responseWriter.WriteHeader(code)
}

// Get the response header.
func (c *context) Header() http.Header {
	return c.responseWriter.Header()
}

// Get Default format error, which is like:
//
//     type Error struct {
//         Code    int
//         Message string
//     }
//
// And it will marshal to special mime-type when calling with Service.Error.
func (c *context) DetailError(code int, format string, args ...interface{}) error {
	marshaller, ok := getMarshaller(c.mime)
	if !ok {
		http.Error(c.responseWriter, "can't find marshaller for"+c.mime, http.StatusBadRequest)
		return errors.New("can't find marshaller for" + c.mime)
	}
	return marshaller.Error(code, fmt.Sprintf(format, args...))
}

// Error replies to the request with the specified error message and HTTP code.
// If err has export field, it will be marshalled to response.Body directly, otherwise will use err.Error().
func (c *context) Error(code int, err error) {
	c.WriteHeader(code)
	marshaller, ok := getMarshaller(c.mime)
	if !ok {
		http.Error(c.responseWriter, "can't find marshaller for"+c.mime, http.StatusBadRequest)
		return
	}
	if hasExportField(err) {
		marshaller.Marshal(c.responseWriter, c.name, err)
	} else {
		marshaller.Marshal(c.responseWriter, c.name, err.Error())
	}
	c.isError = true
}

// Redirect to the specified path.
func (c *context) RedirectTo(path string) {
	c.Header().Set("Location", path)
	c.WriteHeader(http.StatusTemporaryRedirect)
}

func hasExportField(i interface{}) bool {
	v := reflect.ValueOf(i)
	v = reflect.Indirect(v)
	t := v.Type()
	for i, n := 0, t.NumField(); i < n; i++ {
		name := t.Field(i).Name
		if c := name[0]; 'A' <= c && c <= 'Z' {
			return true
		}
	}
	return false
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
