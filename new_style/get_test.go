package rest

import (
	"github.com/googollee/go-assert"
	"reflect"
	"testing"
)

func GetOKFunc(ctx Context) (int, error)               { return 1, nil }
func NoInputGetFunc() (int, error)                     { return 1, nil }
func NoContextGetFunc(i int) (int, error)              { return 1, nil }
func MoreInputGetFunc(ctx Context, i int) (int, error) { return 1, nil }
func NoOutputGetFunc(ctx Context)                      {}
func NoErrorGetFunc(ctx Context) int                   { return 1 }
func MoreOutputGetFunc(ctx Context) (int, error, int)  { return 1, nil, 1 }

func TestGetInit(t *testing.T) {
	type Test struct {
		f  reflect.Value
		ok bool
	}
	var tests = []Test{
		{reflect.ValueOf(GetOKFunc), true},

		{reflect.ValueOf(1), false},
		{reflect.ValueOf(NoInputGetFunc), false},
		{reflect.ValueOf(NoContextGetFunc), false},
		{reflect.ValueOf(MoreInputGetFunc), false},
		{reflect.ValueOf(NoOutputGetFunc), false},
		{reflect.ValueOf(NoErrorGetFunc), false},
		{reflect.ValueOf(MoreOutputGetFunc), false},
	}
	for i, test := range tests {
		get := Get{}
		err := get.init(test.f)
		assert.Equal(t, err == nil, test.ok, "test %d: %s", i, err)
	}
}
