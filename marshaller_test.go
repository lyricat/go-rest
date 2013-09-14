package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"io"
	"testing"
)

type FailMarshaller struct{}

func (m FailMarshaller) Marshal(w io.Writer, name string, v interface{}) error {
	return fmt.Errorf("failed")
}

func (m FailMarshaller) Unmarshal(r io.Reader, v interface{}) error {
	return fmt.Errorf("failed")
}

type FakeMarshaller struct {
}

func (m FakeMarshaller) Marshal(w io.Writer, name string, v interface{}) error {
	_, err := w.Write([]byte(fmt.Sprintf("fake marshal writed: %#v", v)))
	return err
}

func (m FakeMarshaller) Unmarshal(r io.Reader, v interface{}) error {
	return nil
}

func TestGetMarshallerConcurrency(t *testing.T) {
	RegisterMarshaller("text/fake", FakeMarshaller{})
	n := 100
	quit := make(chan int)
	for i := 0; i < n; i++ {
		mime := "text/fake"
		if i%3 == 0 {
			mime = "application/json"
		} else if i%3 == 1 {
			mime = "non/exist"
		}
		go func(mime string) {
			_, ok := getMarshaller(mime)
			if mime == "non/exist" {
				assert.Equal(t, ok, false)
			} else {
				assert.Equal(t, ok, true)
			}
			quit <- 1
		}(mime)
	}
	for i := 0; i < n; i++ {
		<-quit
	}
}
