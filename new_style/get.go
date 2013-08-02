package rest

import (
	"errors"
	"reflect"
)

type Get struct {
	f reflect.Value
}

func (g *Get) init(f reflect.Value) error {
	if f.Kind() != reflect.Func {
		return errors.New("not function")
	}
	t := f.Type()
	if t.NumIn() != 1 || t.In(0) != typeOfContext {
		return errors.New("GET handler must has 1 input parameter: Context")
	}
	if t.NumOut() != 2 || t.Out(1) != typeOfError {
		return errors.New("GET handler 2nd output parameter must be error")
	}

	g.f = f
	return nil
}

func (g *Get) method() string {
	return "GET"
}

func (g *Get) handle(ctx Context) {
	rets := g.f.Call([]reflect.Value{reflect.ValueOf(ctx)})
	v, err := rets[0].Interface(), rets[1].Interface()
	if err != nil {
		ctx.handleError(err.(error))
		return
	}
	ctx.responseCode(OK)
	ctx.response(v)
}
