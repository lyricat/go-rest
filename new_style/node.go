package rest

import (
	"reflect"
)

type handler interface {
	init(f reflect.Value) error
	method() string
	handle(ctx Context)
}

var handlerType = reflect.TypeOf((*handler)(nil)).Elem()

type namedNode struct {
	name    string
	type_   string
	handler handler
}

type node struct {
	path     string
	methods  string
	handlers map[string]*namedNode
}
