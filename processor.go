package rest

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type innerProcessor struct {
	pathFormatter string
	responseType  reflect.Type
	funcIndex     int
}

/*
Define the process.

Valid tag:

 - method: Define the method of http request.
 - path: Define the path of http request.
 - func: Define the corresponding function name.
 - mime: Define the default mime of request's and response's body. It overwrite the service one.

To be implement:
 - charset: Define the default charset of request's and response's body. It overwrite the service one.
 - scope: Define required scope when process.
*/
type Processor struct {
	*innerProcessor
}

// Generate the path of http request to processor. The args will fill in by url order.
func (p Processor) Path(args ...interface{}) (string, error) {
	return fmt.Sprintf(p.pathFormatter, args...), nil
}

func (p Processor) init(processor reflect.Value, pathFormatter string, f reflect.Method, tag reflect.StructTag) error {
	retType, err := parseResponseType(f.Type)
	if err != nil {
		return err
	}

	processor.Field(0).Set(reflect.ValueOf(&innerProcessor{
		pathFormatter: pathFormatter,
		responseType:  retType,
		funcIndex:     f.Index,
	}))

	return nil
}

func (p Processor) handle(instance reflect.Value, ctx *context, args []reflect.Value) {
	w := ctx.responseWriter
	marshaller := ctx.marshaller
	f := instance.Method(p.funcIndex)
	ret := f.Call(args)

	w.WriteHeader(ctx.response.Status)
	if 200 <= ctx.response.Status && ctx.response.Status <= 399 && len(ret) > 0 {
		marshaller.Marshal(w, ret[0].Interface())
	} else if ctx.error != nil {
		w.Write([]byte(ctx.error.Error()))
	}
}

func parseResponseType(f reflect.Type) (ret reflect.Type, err error) {
	if f.NumOut() == 0 {
		return
	}
	if f.NumOut() > 1 {
		err = fmt.Errorf("processor(%s) return more than 1 value", f.Name())
		return
	}
	ret = f.Out(0)
	if !checkTypeCanMarshal(ret) {
		err = fmt.Errorf("processor(%s)'s return type %s can't be marshaled", f.Name(), ret.String())
	}
	return
}

func checkTypeCanMarshal(t reflect.Type) bool {
	val := reflect.New(t)
	null := new(nullWriter)
	encoder := json.NewEncoder(null)
	return encoder.Encode(val.Interface()) == nil
}

type nullWriter struct{}

func (w nullWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
