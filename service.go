package rest

import (
	"reflect"
)

/*
Define the service.

Valid tag:

 - prefix: The prefix path of http request. All processor's path will prefix with prefix path.
 - mime: Define the default mime of all processor in this service.
 - compress: If value is "on", it will compress response using "Accept-Encoding" in request header.

To be implement:
 - charset: Define the default charset of all processor in this service.
*/
type Service struct {
	*context
}

func initService(service reflect.Value, tag reflect.StructTag) (string, string, string, error) {
	mime := tag.Get("mime")
	if mime == "" {
		mime = "application/json"
	}

	charset := tag.Get("charset")
	if charset == "" {
		charset = "utf-8"
	}

	prefix := tag.Get("prefix")
	if prefix == "" {
		prefix = "/"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if l := len(prefix); l > 2 && prefix[l-1] == '/' {
		prefix = prefix[:l-1]
	}

	return prefix, mime, charset, nil
}
