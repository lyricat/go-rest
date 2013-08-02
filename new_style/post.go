package rest

import (
	"errors"
	"reflect"
)

type Post struct {
	f           reflect.Value
	request     reflect.Type
	requestPtr  bool
	hasResponse bool
}

func (p *Post) init(f reflect.Value) error {
	if f.Kind() != reflect.Func {
		return errors.New("not function")
	}
	t := f.Type()
	if t.NumIn() != 2 || t.In(0) != typeOfContext {
		return errors.New("POST handler must has 1 input parameter: Context")
	}
	if t.NumOut() != 1 && t.NumOut() != 2 {
		return errors.New("POST handler should return one or two output")
	}
	if t.Out(t.NumOut()-1) != typeOfError {
		return errors.New("POST handler last output parameter must be error")
	}

	p.f = f
	if t.In(1).Kind() == reflect.Ptr {
		p.request = t.In(1).Elem()
		p.requestPtr = true
	} else {
		p.request = t.In(1)
		p.requestPtr = false
	}
	p.hasResponse = t.NumOut() == 2
	return nil
}

func (p *Post) method() string {
	return "POST"
}

func (p *Post) handle(ctx Context) {
	marshaller, err := ctx.marshaller()
	if err != nil {
		return
	}

	req := reflect.New(p.request)
	err = marshaller.Unmarshal(ctx.Body(), req.Interface())
	if err != nil {
		ctx.handleError(BadRequest(err))
		return
	}
	if !p.requestPtr {
		req = req.Elem()
	}

	rets := p.f.Call([]reflect.Value{reflect.ValueOf(ctx), req})

	e := rets[len(rets)-1].Interface()
	if e != nil {
		ctx.handleError(e.(error))
		return
	}
	ctx.responseCode(Accepted)
	if p.hasResponse {
		v := rets[0].Interface()
		ctx.response(v)
	}
}
