package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
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
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			t.Error(err)
			continue
		}

		assert.Equal(t, service.Prefix, test.prefix, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultMime, test.mime, fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultCharset, test.charset, fmt.Sprintf("test %d", i))

		assert.Equal(t, prefix, test.prefix, fmt.Sprintf("test %d", i))
		assert.Equal(t, mime, test.mime, fmt.Sprintf("test %d", i))
		assert.Equal(t, charset, test.charset, fmt.Sprintf("test %d", i))
	}

	// overwrite tag
	for i, test := range tests {
		service := new(Service)
		service.Prefix = "/abcde"
		service.DefaultMime = "text/plain"
		service.DefaultCharset = "abc"
		prefix, mime, charset, err := initService(reflect.ValueOf(service).Elem(), test.tag)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d", i))
		if !test.ok {
			t.Error(err)
			continue
		}

		assert.Equal(t, service.Prefix, "/abcde", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultMime, "text/plain", fmt.Sprintf("test %d", i))
		assert.Equal(t, service.DefaultCharset, "abc", fmt.Sprintf("test %d", i))

		assert.Equal(t, prefix, "/abcde", fmt.Sprintf("test %d", i))
		assert.Equal(t, mime, "text/plain", fmt.Sprintf("test %d", i))
		assert.Equal(t, charset, "abc", fmt.Sprintf("test %d", i))
	}
}
