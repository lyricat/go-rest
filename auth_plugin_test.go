package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestCheckScope(t *testing.T) {
	type Test struct {
		userScopes []string
		tag        reflect.StructTag
		ok         bool
		lack       string
	}
	var tests = []Test{
		{[]string{"a"}, `scope:"*"`, true, ""},
		{[]string{"a"}, `scope:"a"`, true, ""},
		{[]string{"a", "b"}, `scope:"a"`, true, ""},
		{[]string{"a"}, `scope:"a,b"`, false, "b"},
	}

	p := NewAuthPlugin(getScope)
	for i, test := range tests {
		ok, lack := p.checkScope(test.userScopes, test.tag)
		assert.Equal(t, ok, test.ok, fmt.Sprintf("test %d", i))
		assert.Equal(t, lack, test.lack, fmt.Sprintf("test %d", i))
	}
}

func TestPreProcessor(t *testing.T) {
	type Test struct {
		userScopes   string
		serviceTag   reflect.StructTag
		processorTag reflect.StructTag
		pass         bool
	}
	var tests = []Test{
		{"a", `scope:"*"`, `scope:"*"`, true},
		{"a", `scope:"a"`, `scope:"a"`, true},
		{"a", `scope:"a,b"`, `scope:"*"`, false},
		{"a", `scope:"a"`, `scope:"a,b"`, false},
		{"a,b", `scope:"a,b"`, `scope:"a"`, true},
	}

	p := NewAuthPlugin(getScope)
	for i, test := range tests {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Scope", test.userScopes)
		service := Service{}
		service.Tag = test.serviceTag
		processor := Processor{}
		processor.Tag = test.processorTag
		resp := p.PreProcess(req, service, processor)
		assert.Equal(t, resp == nil, test.pass, fmt.Sprintf("test %d", i))
		if resp != nil {
			assert.Equal(t, resp.Status, http.StatusUnauthorized, fmt.Sprintf("test %d", i))

		}
	}
}

func getScope(r *http.Request) []string {
	return strings.Split(r.Header.Get("Scope"), ",")
}
