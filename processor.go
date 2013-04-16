package rest

import (
	"fmt"
	"net/http"
	"reflect"
)

/*
Define the processor to handle normal http request. It should return immediately.

The processor's handle function may take 0 or 1 input parameter which unmashal from request body,
and return 0 or 1 value for response body, like below:

 - func Handler() // ignore request body, no response
 - func Handler(post PostType) // marshal request to PostType, no response
 - func Hanlder() ResponseType // ignore request body, response type is ResponseType
 - func Handler(post PostType) ResponseType // marshal request to PostType, response type is ResponseType

If function's input nothing, processor will let function to handle request's body directly through
Service.Request().

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

// Generate the path of url to processor. Map args fill parameters in path.
func (p Processor) PathMap(args map[string]string) string {
	return p.formatter.pathMap(args)
}

// Generate the path of url to processor. It accepts a sequence of key/value pairs, and fill parameters in path.
func (p Processor) Path(args ...string) string {
	return p.formatter.path(args...)
}

type innerProcessor struct {
	formatter    pathFormatter
	requestType  reflect.Type
	responseType reflect.Type
	funcIndex    int
}

func (i *innerProcessor) init(formatter pathFormatter, f reflect.Method, tag reflect.StructTag) error {
	ft := f.Type
	if ft.NumIn() > 2 {
		return fmt.Errorf("processer(%s) input parameters should be no more than 2.", f.Name)
	}
	if ft.NumIn() == 2 {
		i.requestType = ft.In(1)
	}

	if ft.NumOut() > 1 {
		return fmt.Errorf("processor(%s) return should be no more than 1 value.", f.Name)
	}
	if ft.NumOut() == 1 {
		i.responseType = ft.Out(0)
	}

	i.formatter = formatter
	i.funcIndex = f.Index

	return nil
}

func (i *innerProcessor) handle(instance reflect.Value, ctx *context) {
	r := ctx.request
	w := ctx.responseWriter
	marshaller := ctx.marshaller
	f := instance.Method(i.funcIndex)
	var args []reflect.Value

	if i.requestType != nil {
		request := reflect.New(i.requestType)
		err := marshaller.Unmarshal(r.Body, request.Interface())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		args = append(args, request.Elem())
	}
	ret := f.Call(args)

	if !ctx.isError && len(ret) > 0 {
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=%s", ctx.mime, ctx.charset))
		err := marshaller.Marshal(w, ret[0].Interface())
		if err != nil {
			w.Write([]byte(err.Error()))
		}
	}
}
