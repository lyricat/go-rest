package rest

import (
	"github.com/googollee/go-assert"
	"testing"
)

func TestSandardError(t *testing.T) {
	type Test struct {
		code      int
		args      []interface{}
		errString string
	}
	var tests = []Test{
		{1, nil, ""},
		{1, []interface{}{1}, "1"},
		{1, []interface{}{"abc"}, "abc"},
		{1, []interface{}{"%s,%d", "abc", 1}, "abc,1"},
	}
	for i, test := range tests {
		err := NewStandardError(test.code, test.args...)
		assert.Equal(t, err.Error(), test.errString, "test %d", i)
	}
}

func TestRawPathToType(t *testing.T) {
	type Test struct {
		rawPath string
		type_   string
	}
	var tests = []Test{
		{"/", "/"},
		{"/prefix", "/prefix"},
		{"/prefix/obj/:id", "/prefix/obj"},
		{"/prefix/obj/:id/other", "/prefix/obj/other"},
		{"/prefix/obj/:id/other/:oid", "/prefix/obj/other"},
		{"/prefix/obj/*exta", "/prefix/obj"},
	}
	for i, test := range tests {
		type_ := rawPathToType(test.rawPath)
		assert.Equal(t, type_, test.type_, "test %d", i)
	}
}
