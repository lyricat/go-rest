package rest

import (
	"fmt"
	"net/http"
	"testing"
)

func TestNewContext(t *testing.T) {
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
		defaultMime    string
		defaultCharset string

		ok         bool
		mime       string
		charset    string
		compresser Compresser
		response   http.ResponseWriter
	}
	var tests = []Test{
		{map[string]string{"Accept": "application/json", "Accept-Charset": "utf-8"}, "application/json", "utf-8", true, "application/json", "utf-8", nil, nil},
		{map[string]string{"Content-Type": "application/json", "Accept": "application/json", "Accept-Charset": "utf-8"}, "application/xml", "gbk", true, "application/json", "utf-8", nil, nil},
		{nil, "application/json", "utf-8", true, "application/json", "utf-8", nil, nil},
		{map[string]string{"Accept": "application/unknown", "Accept-Charset": "utf-8"}, "application/json", "utf-8", true, "application/json", "utf-8", nil, nil},
		{map[string]string{"Accept": "application/unknown", "Accept-Charset": "utf-8"}, "application/unknow", "utf-8", false, "", "", nil, nil},

		{map[string]string{"Accept-Encoding": "gzip"}, "application/json", "utf-8", true, "application/json", "utf-8", gzip, nil},
		{map[string]string{"Accept-Encoding": "deflate"}, "application/json", "utf-8", true, "application/json", "utf-8", flate, nil},
		{map[string]string{"Accept-Encoding": "gzip, deflate"}, "application/json", "utf-8", true, "application/json", "utf-8", gzip, nil},
		{map[string]string{"Accept-Encoding": "unknown, gzip"}, "application/json", "utf-8", true, "application/json", "utf-8", gzip, nil},
		{map[string]string{"Accept-Encoding": "unknown"}, "application/json", "utf-8", true, "application/json", "utf-8", nil, nil},
	}
	for i, test := range tests {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal("invalid request")
		}
		for k, v := range test.headers {
			req.Header.Set(k, v)
		}
		ctx, err := newContext(nil, req, nil, test.defaultMime, test.defaultCharset)
		equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			continue
		}
		equal(t, ctx.mime, test.mime, fmt.Sprintf("test %d", i))
		equal(t, ctx.charset, test.charset, fmt.Sprintf("test %d", i))
		equal(t, ctx.request, req, fmt.Sprintf("test %d", i))
		equal(t, ctx.responseWriter, test.response, fmt.Sprintf("test %d", i))
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
		equal(t, ret, test.ret, fmt.Sprintf("test %d", i))
		if !equalMap(pair, test.pair) {
			t.Errorf("test %d not equal:\nexpect: %v\ngot: %v", i, test.pair, pair)
		}
	}
}

func TestHasExportField(t *testing.T) {
	type Test struct {
		i  interface{}
		ok bool
	}
	type NoExport struct {
		i int
	}
	type NoField struct{}
	type HasExport struct {
		I int
	}
	type ComposeExport struct {
		HasExport
	}
	var tests = []Test{
		{&NoExport{}, false},
		{&NoField{}, false},
		{&HasExport{}, true},
		{&ComposeExport{}, true},
		{NoExport{}, false},
		{NoField{}, false},
		{HasExport{}, true},
		{ComposeExport{}, true},
	}
	for i, test := range tests {
		ok := hasExportField(test.i)
		equal(t, ok, test.ok, "test %d", i)
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
