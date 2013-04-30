package rest

import (
	"encoding/json"
	"fmt"
	"io"
)

type Marshaller interface {
	Marshal(w io.Writer, v interface{}) error
	Unmarshal(r io.Reader, v interface{}) error
	Error(code int, message string) error
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

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e jsonError) Error() string {
	return fmt.Sprintf("(%d)%s", e.Code, e.Message)
}

func (j JsonMarshaller) Error(code int, message string) error {
	return jsonError{code, message}
}
