package rest

import (
	"fmt"
	"reflect"
	"testing"
)

func TestService(t *testing.T) {
	type Test struct {
		tag reflect.StructTag

		ok      bool
		prefix  string
		mime    string
		charset string
	}
	var tests = []Test{
		{``, true, "/", "application/json", "utf-8"},
		{`prefix:"/prefix" realm:"abc,xyz" mime:"application/xml" charset:"gbk"`, true, "/prefix", "application/xml", "gbk"},
		{`prefix:"/prefix" realm:"abc,xyz" charset:"gbk"`, true, "/prefix", "application/json", "gbk"},
		{`prefix:"/prefix" realm:"abc,xyz" mime:"application/xml"`, true, "/prefix", "application/xml", "utf-8"},
		{`realm:"abc,xyz" mime:"application/xml"`, true, "/", "application/xml", "utf-8"},
	}

	for i, test := range tests {
		service := new(Service)
		prefix, mime, charset, err := initService(reflect.ValueOf(service).Elem(), test.tag)
		equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			t.Error(err)
			continue
		}

		equal(t, prefix, test.prefix, fmt.Sprintf("test %d", i))
		equal(t, mime, test.mime, fmt.Sprintf("test %d", i))
		equal(t, charset, test.charset, fmt.Sprintf("test %d", i))
	}
}
