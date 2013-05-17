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
	headerWriter   headerWriter
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
