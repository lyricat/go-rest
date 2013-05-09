package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"net/http"
	"testing"
)

func TestNewContext(t *testing.T) {
	json, ok := getMarshaller("application/json")
	if !ok {
		t.Fatalf("can't find json marshaller")
	}
	flate, ok := getCompresser("deflate")
	if !ok {
		t.Fatalf("can't find deflate compresser")
	}
	gzip, ok := getCompresser("gzip")
	if !ok {
		t.Fatalf("can't find gzip compresser")
	}
	type Test struct {
		headers        map[string]string
		w              http.ResponseWriter
		defaultMime    string
		defaultCharset string

		ok         bool
		mime       string
		charset    string
		marshaller Marshaller
		compresser Compresser
		response   http.ResponseWriter
	}
	var tests = []Test{
		{map[string]string{"Accept": "application/json", "Accept-Charset": "utf-8"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, nil, nil},
		{map[string]string{"Accept": "application/json", "Accept-Charset": "utf-8"}, nil, "application/xml", "gbk", true, "application/json", "utf-8", json, nil, nil},
		{nil, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, nil, nil},
		{map[string]string{"Accept": "application/unknown", "Accept-Charset": "utf-8"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, nil, nil},
		{map[string]string{"Accept": "application/unknown", "Accept-Charset": "utf-8"}, nil, "application/unknow", "utf-8", false, "", "", nil, nil, nil},

		{map[string]string{"Accept-Encoding": "gzip"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, gzip, nil},
		{map[string]string{"Accept-Encoding": "deflate"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, flate, nil},
		{map[string]string{"Accept-Encoding": "gzip, deflate"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, gzip, nil},
		{map[string]string{"Accept-Encoding": "unknown, gzip"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, gzip, nil},
		{map[string]string{"Accept-Encoding": "unknown"}, nil, "application/json", "utf-8", true, "application/json", "utf-8", json, nil, nil},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal("invalid request")
		}
		for k, v := range test.headers {
			req.Header.Set(k, v)
		}
		ctx, err := newContext(test.w, req, nil, test.defaultMime, test.defaultCharset)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			continue
		}
		assert.Equal(t, ctx.mime, test.mime, fmt.Sprintf("test %d", i))
		assert.Equal(t, ctx.charset, test.charset, fmt.Sprintf("test %d", i))
		assert.Equal(t, ctx.marshaller, test.marshaller, fmt.Sprintf("test %d", i))
		assert.Equal(t, ctx.request, req, fmt.Sprintf("test %d", i))
		assert.Equal(t, ctx.responseWriter, test.response, fmt.Sprintf("test %d", i))
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
		assert.Equal(t, ret, test.ret, fmt.Sprintf("test %d", i))
		if !equalMap(pair, test.pair) {
			t.Errorf("test %d not equal:\nexpect: %v\ngot: %v", i, test.pair, pair)
		}
	}
}

func equalMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if vb, ok := b[k]; !ok || vb != v {
			return false
		}
	}
	return true
}
