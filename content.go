package rest

import (
	"fmt"
	"net/http"
	"strings"
)

type context struct {
	mime           string
	marshaller     Marshaller
	request        *http.Request
	responseWriter http.ResponseWriter
	isError        bool
}

func newContent(w http.ResponseWriter, r *http.Request, defaultMime, defaultCharset string) (*context, error) {
	mime, charset := getContentTypeFromRequset(r)
	if mime == "" {
		mime = defaultMime
	}
	if charset == "" {
		charset = defaultCharset
	}

	marshaller, ok := getMarshaller(mime)
	if !ok {
		mime = defaultMime
		marshaller, ok = getMarshaller(mime)
	}
	if !ok {
		return nil, fmt.Errorf("can't find %s marshaller", mime)
	}

	return &context{
		mime:           mime,
		marshaller:     marshaller,
		request:        r,
		responseWriter: w,
		isError:        false,
	}, nil
}

func getContentTypeFromRequset(r *http.Request) (string, string) {
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

	return mime, charset
}
