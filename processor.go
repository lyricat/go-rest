package rest

import (
	"fmt"
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
*/
type Processor struct {
	formatter pathFormatter
}

// Generate the path of url to processor. Map args fill parameters in path.
func (p Processor) PathMap(args map[string]string) string {
	return p.formatter.pathMap(args)
}

// Generate the path of url to processor. It accepts a sequence of key/value pairs, and fill parameters in path.
func (p Processor) Path(args ...string) string {
	return p.formatter.path(args...)
}

func (p *Processor) init(formatter pathFormatter, instance reflect.Type, name string, tag reflect.StructTag) ([]handler, []pathFormatter, error) {
	fname := tag.Get("func")
	if fname == "" {
		fname = "Handle" + name
	}
	f, ok := instance.MethodByName(fname)
	if !ok {
		return nil, nil, fmt.Errorf("can't find handler: %s", fname)
	}

	ft := f.Type
	ret := new(processorNode)
	ret.funcIndex = f.Index
	if ft.NumIn() > 2 {
		return nil, nil, fmt.Errorf("processer(%s) input parameters should be no more than 2.", f.Name)
	}
	if ft.NumIn() == 2 {
		ret.requestType = ft.In(1)
	}

	if ft.NumOut() > 1 {
		return nil, nil, fmt.Errorf("processor(%s) return should be no more than 1 value.", f.Name)
	}
	if ft.NumOut() == 1 {
		ret.responseType = ft.Out(0)
	}

	p.formatter = formatter

	return []handler{ret}, []pathFormatter{formatter}, nil
}
