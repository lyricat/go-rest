package rest

import (
	"fmt"
	"net/http"
	"strings"
)

type headerWriter interface {
	writeHeader(int)
}

type context struct {
	mime           string
	charset        string
	marshaller     Marshaller
	compresser     Compresser
	vars           map[string]string
	request        *http.Request
	responseWriter http.ResponseWriter
	isError        bool
}

func newContext(w http.ResponseWriter, r *http.Request, vars map[string]string, defaultMime, defaultCharset string) (*context, error) {
	mime := r.Header.Get("Accept")
	charset := r.Header.Get("Accept-Charset")
	if mime == "" {
		mime = defaultMime
	}
	if charset == "" {
		charset = defaultCharset
	}
	if charset == "" {
		charset = "utf-8"
	}

	marshaller, ok := getMarshaller(mime)
	if !ok {
		mime = defaultMime
		marshaller, ok = getMarshaller(mime)
	}
	if !ok {
		return nil, fmt.Errorf("can't find %s marshaller", mime)
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
		mime:           mime,
		charset:        charset,
		marshaller:     marshaller,
		compresser:     compresser,
		vars:           vars,
		request:        r,
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
	return c.marshaller.Error(code, fmt.Sprintf(format, args...))
}

// Error replies to the request with the specified error message and HTTP code.
func (c *context) Error(code int, err error) {
	c.WriteHeader(code)
	c.marshaller.Marshal(c.responseWriter, err)
	c.isError = true
}

// Redirect to the specified path.
func (c *context) RedirectTo(path string) {
	c.Header().Set("Location", path)
	c.WriteHeader(http.StatusTemporaryRedirect)
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
