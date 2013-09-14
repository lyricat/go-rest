package rest

import (
	"fmt"
	"github.com/googollee/go-assert"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestUpperFirst(t *testing.T) {
	type Test struct {
		i string
		o string
	}
	var tests = []Test{
		{"Abc", "Abc"},
		{"abc", "Abc"},
	}
	for i, test := range tests {
		assert.Equal(t, upperFirst(test.i), test.o, "test %d", i)
	}
}

func TestParseHeaderField(t *testing.T) {
	type Test struct {
		header string
		field  string
		ret    string
		pair   map[string]string
	}
	var tests = []Test{
		{"", "Abc", "", nil},
		{"text/plain", "Accept", "text/plain", nil},
		{"text/plain; charset=utf8", "Content-Type", "text/plain", map[string]string{"charset": "utf8"}},
		{"text/plain; charset=utf8;", "Content-Type", "text/plain", map[string]string{"charset": "utf8"}},
		{"text/plain; charset", "Content-Type", "text/plain", map[string]string{"charset": ""}},
		{"text/plain; charset=utf8; skin=new", "Content-Type", "text/plain", map[string]string{"charset": "utf8", "skin": "new"}},
	}

	for i, test := range tests {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal("invalid request")
		}
		req.Header.Set(test.field, test.header)
		ret, pair := parseHeaderField(req, test.field)
		assert.Equal(t, ret, test.ret, "test %d", i)
		assert.Equal(t, pair, test.pair, "test %d not equal:\nexpect: %v\ngot: %v", i, test.pair, pair)
	}
}

func TestGetMarshallerFromRequest(t *testing.T) {
	fakeMarshaller := FakeMarshaller{}
	RegisterMarshaller("fake/mime", fakeMarshaller)
	type Test struct {
		mime             string
		marshaller       Marshaller
		header           http.Header
		targetMime       string
		targetMarshaller Marshaller
	}
	var tests = []Test{
		{"", nil, nil, "application/json", jsonMarshaller},
		{"fake/mime", fakeMarshaller, nil, "fake/mime", fakeMarshaller},
		{"fake/mime", fakeMarshaller, map[string][]string{"Content-Type": []string{"application/json"}}, "application/json", jsonMarshaller},
		{"fake/mime", fakeMarshaller, map[string][]string{"Content-Type": []string{"application/json; charset=utf-8"}}, "application/json", jsonMarshaller},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "http://domain/", nil)
		assert.MustEqual(t, err, nil, "test %d", i)
		req.Header = test.header
		mime, marshaller := getMarshallerFromRequest(test.mime, test.marshaller, req)
		assert.Equal(t, mime, test.targetMime, "test %d", i)
		assert.Equal(t, marshaller, test.targetMarshaller, "test %d", i)
	}
}

func TestUnmashallFromReader(t *testing.T) {
	type data struct {
		ID int `json:"id"`
	}
	type Test struct {
		t          reflect.Type
		marshaller Marshaller
		r          io.Reader
		value      string
		ok         bool
	}
	var tests = []Test{
		{reflect.TypeOf(1), jsonMarshaller, strings.NewReader(`1`), "1", true},
		{reflect.TypeOf(""), jsonMarshaller, strings.NewReader(`"str"`), `"str"`, true},
		{reflect.TypeOf(data{}), jsonMarshaller, strings.NewReader(`{"id":1}`), `rest.data{ID:1}`, true},
		{reflect.TypeOf(&data{}), jsonMarshaller, strings.NewReader(`{"id":1}`), `&rest.data{ID:1}`, true},
		{reflect.TypeOf(&data{}), jsonMarshaller, strings.NewReader(`1`), ``, false},
	}
	for i, test := range tests {
		v, err := unmarshallFromReader(test.t, test.marshaller, test.r)
		assert.MustEqual(t, err == nil, test.ok, "test %d", i)
		if err != nil {
			continue
		}
		assert.Equal(t, fmt.Sprintf("%#v", v.Interface()), test.value, "test %d", i)
	}
}
