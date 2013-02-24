package rest

import (
	"encoding/json"
	"io"
)

type Marshaller interface {
	Marshal(w io.Writer, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
}

// Register a marshaller with corresponding mime.
func RegisterMarshaller(mime string, marshaller Marshaller) {
	marshallers[mime] = marshaller
}

var marshallers map[string]Marshaller

func init() {
	marshallers = map[string]Marshaller{
		"application/json": new(JsonMarshaller),
	}
}

func getMarshaller(mime string) (Marshaller, bool) {
	ret, ok := marshallers[mime]
	return ret, ok
}

// The marshaller using json.
type JsonMarshaller struct{}

func (j JsonMarshaller) Marshal(w io.Writer, v interface{}) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(v)
}

func (j JsonMarshaller) Unmarshal(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}
