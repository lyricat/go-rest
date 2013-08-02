package rest

import (
	"github.com/googollee/go-assert"
	"reflect"
	"testing"
)

func PostOKFunc1(ctx Context, i int) (int, error)             { return 1, nil }
func PostOKFunc2(ctx Context, i int) error                    { return nil }
func PostOKFunc3(ctx Context, i *int) error                   { return nil }
func NoInputPostFunc1() (int, error)                          { return 1, nil }
func NoInputPostFunc2(ctx Context) (int, error)               { return 1, nil }
func NoContextPostFunc(i int) (int, error)                    { return 1, nil }
func MoreInputPostFunc(ctx Context, i, j int) (int, error)    { return 1, nil }
func NoOutputPostFunc(ctx Context, i int)                     {}
func NoErrorPostFunc(ctx Context, i int) int                  { return 1 }
func MoreOutputPostFunc(ctx Context, i int) (int, error, int) { return 1, nil, 1 }

func TestPostInit(t *testing.T) {
	type Test struct {
		f           reflect.Value
		ok          bool
		isPtr       bool
		request     string
		hasResponse bool
	}
	var tests = []Test{
		{reflect.ValueOf(PostOKFunc1), true, false, "int", true},
		{reflect.ValueOf(PostOKFunc2), true, false, "int", false},
		{reflect.ValueOf(PostOKFunc3), true, true, "int", false},

		{reflect.ValueOf(1), false, false, "", false},
		{reflect.ValueOf(NoInputPostFunc1), false, false, "", false},
		{reflect.ValueOf(NoInputPostFunc2), false, false, "", false},
		{reflect.ValueOf(NoContextPostFunc), false, false, "", false},
		{reflect.ValueOf(MoreInputPostFunc), false, false, "", false},
		{reflect.ValueOf(NoOutputPostFunc), false, false, "", false},
		{reflect.ValueOf(NoErrorPostFunc), false, false, "", false},
		{reflect.ValueOf(MoreOutputPostFunc), false, false, "", false},
	}
	for i, test := range tests {
		post := Post{}
		err := post.init(test.f)
		assert.MustEqual(t, err == nil, test.ok, "test %d: %s", i, err)
		if err != nil {
			continue
		}
		assert.Equal(t, post.requestPtr, test.isPtr, "test %d", i)
		assert.Equal(t, post.request.String(), test.request, "test %d", i)
		assert.Equal(t, post.hasResponse, test.hasResponse, "test %d", i)
	}
}
