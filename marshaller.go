package rest

import (
	"encoding/json"
	"io"
)

// Marshaller is a mime type marshaller.
type Marshaller interface {
	Marshal(w io.Writer, name string, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
}

// RegisterMarshaller register a marshaller with corresponding mime.
func RegisterMarshaller(mime string, marshaller Marshaller) {
	marshallers[mime] = marshaller
}

var jsonMarshaller = JSONMarshaller{}
var marshallers = map[string]Marshaller{
	"application/json": jsonMarshaller,
}

func getMarshaller(mime string) (Marshaller, bool) {
	ret, ok := marshallers[mime]
	return ret, ok
}

// JSONMarshaller is Marshaller using json.
type JSONMarshaller struct{}

// Marshal will marshal v and write to w, with the handler function name.
func (j JSONMarshaller) Marshal(w io.Writer, name string, v interface{}) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(v)
}

// Unmarshal will read r and unmarshal to v.
func (j JSONMarshaller) Unmarshal(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(v)
}
