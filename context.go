package rest

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Context is information about a http request/response.
type Context interface {
	// Request of http process.
	Request() *http.Request

	// Response of http process.
	Response() http.ResponseWriter

	// Bind parameter id of url's query/form/path to v.
	// It will convert parameter to following data type automatically:
	//  - bool
	//  - string and array of string
	//  - int of all widths and array of int
	//  - float of all width and array of float
	// If converting error, check Context.BindError() when all bind finished.
	Bind(id string, v interface{})

	// BindError return the error when binding parameters.
	BindError() error

	// BindReset reset the error of binding to nil.
	BindReset()

	// Return use code as http response code.
	// If giving fmtAndArgs, it will format to string like fmt.Sprintf(fmtAndArgs...) and use as http response body.
	// Example:
	//     ctx.Return(http.StatusBadRequest, "input error: %s", ctx.BindError())
	Return(code int, fmtAndArgs ...interface{})

	// Render render v as response body, using special marshaller.
	Render(v interface{}) error

	IfMatch(etag string) bool
	IfNoneMatch(etag string) bool
}

type baseContext struct {
	handlerName string
	marshaller  Marshaller
	vars        map[string]string
	request     *http.Request
	response    http.ResponseWriter

	formParsed bool
	bindError  error
}

func newBaseContext(handlerName string, marshaller Marshaller, charset string, vars map[string]string, req *http.Request, resp http.ResponseWriter) *baseContext {
	if marshaller == nil {
		marshaller = jsonMarshaller
	}
	return &baseContext{
		handlerName: handlerName,
		marshaller:  marshaller,
		vars:        vars,
		request:     req,
		response:    resp,

		formParsed: false,
		bindError:  nil,
	}
}

func (ctx *baseContext) IfMatch(etag string) bool {
	return ctx.tagMatch(ctx.request.Header.Get("If-Match"), etag)
}

func (ctx *baseContext) IfNoneMatch(etag string) bool {
	return !ctx.tagMatch(ctx.request.Header.Get("If-None-Match"), etag)
}

func (ctx *baseContext) tagMatch(tags, tag string) bool {
	if tags == "*" {
		return true
	}
	i := strings.Index(tags, tag)
	return i >= 0
}

func (ctx *baseContext) Request() *http.Request {
	return ctx.request
}

func (ctx *baseContext) Response() http.ResponseWriter {
	return ctx.response
}

func (ctx *baseContext) Return(code int, fmtAndArgs ...interface{}) {
	if len(fmtAndArgs) == 0 {
		ctx.response.WriteHeader(code)
		return
	}
	if f, ok := fmtAndArgs[0].(string); ok {
		message := fmt.Sprintf(f, fmtAndArgs[1:]...)
		http.Error(ctx.response, message, code)
		return
	}
	http.Error(ctx.response, fmt.Sprintf("%s", fmtAndArgs[0]), code)
}

func (ctx *baseContext) Render(v interface{}) error {
	return ctx.marshaller.Marshal(ctx.response, ctx.handlerName, v)
}

func (ctx *baseContext) BindError() error {
	return ctx.bindError
}

func (ctx *baseContext) BindReset() {
	ctx.bindError = nil
}

func (ctx *baseContext) Bind(id string, v interface{}) {
	if ctx.bindError != nil {
		return
	}
	switch n := v.(type) {
	case *bool:
		var v []string
		if v, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = v != nil
	case *string:
		*n, ctx.bindError = ctx.getQueryString(id)
	case *int64:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		if *n, ctx.bindError = strconv.ParseInt(v, 10, 64); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int64", id, v)
			return
		}
	case *int:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var i int64
		if i, ctx.bindError = strconv.ParseInt(v, 10, 64); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int", id, v)
			return
		}
		*n = int(i)
	case *int32:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var i int64
		if i, ctx.bindError = strconv.ParseInt(v, 10, 32); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int32", id, v)
			return
		}
		*n = int32(i)
	case *int16:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var i int64
		if i, ctx.bindError = strconv.ParseInt(v, 10, 16); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int16", id, v)
			return
		}
		*n = int16(i)
	case *int8:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var i int64
		if i, ctx.bindError = strconv.ParseInt(v, 10, 8); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int8", id, v)
			return
		}
		*n = int8(i)
	case *uint64:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		if *n, ctx.bindError = strconv.ParseUint(v, 10, 64); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint64", id, v)
			return
		}
	case *uint:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var u uint64
		if u, ctx.bindError = strconv.ParseUint(v, 10, 64); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint", id, v)
			return
		}
		*n = uint(u)
	case *uint32:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var u uint64
		if u, ctx.bindError = strconv.ParseUint(v, 10, 32); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint32", id, v)
			return
		}
		*n = uint32(u)
	case *uint16:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var u uint64
		if u, ctx.bindError = strconv.ParseUint(v, 10, 16); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint16", id, v)
			return
		}
		*n = uint16(u)
	case *uint8:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var u uint64
		if u, ctx.bindError = strconv.ParseUint(v, 10, 8); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint8/byte", id, v)
			return
		}
		*n = uint8(u)
	case *float64:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		if *n, ctx.bindError = strconv.ParseFloat(v, 64); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid float64", id, v)
			return
		}
	case *float32:
		var v string
		if v, ctx.bindError = ctx.getQueryString(id); ctx.bindError != nil {
			return
		}
		var f64 float64
		if f64, ctx.bindError = strconv.ParseFloat(v, 32); ctx.bindError != nil {
			ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid float32", id, v)
			return
		}
		*n = float32(f64)
	case *[]string:
		*n, ctx.bindError = ctx.getQueryStringArray(id)
		return
	case *[]int64:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]int64, len(a))
		for i, v := range a {
			if (*n)[i], ctx.bindError = strconv.ParseInt(v, 10, 64); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int64", id, v)
				return
			}
		}
	case *[]int:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]int, len(a))
		for i, v := range a {
			var i64 int64
			if i64, ctx.bindError = strconv.ParseInt(v, 10, 64); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int", id, v)
				return
			}
			(*n)[i] = int(i64)
		}
	case *[]int32:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]int32, len(a))
		for i, v := range a {
			var i64 int64
			if i64, ctx.bindError = strconv.ParseInt(v, 10, 32); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int32", id, v)
				return
			}
			(*n)[i] = int32(i64)
		}
	case *[]int16:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]int16, len(a))
		for i, v := range a {
			var i64 int64
			if i64, ctx.bindError = strconv.ParseInt(v, 10, 16); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int16", id, v)
				return
			}
			(*n)[i] = int16(i64)
		}
	case *[]int8:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]int8, len(a))
		for i, v := range a {
			var i64 int64
			if i64, ctx.bindError = strconv.ParseInt(v, 10, 8); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid int8", id, v)
				return
			}
			(*n)[i] = int8(i64)
		}
	case *[]uint64:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]uint64, len(a))
		for i, v := range a {
			if (*n)[i], ctx.bindError = strconv.ParseUint(v, 10, 64); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint64", id, v)
				return
			}
		}
	case *[]uint:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]uint, len(a))
		for i, v := range a {
			var u64 uint64
			if u64, ctx.bindError = strconv.ParseUint(v, 10, 64); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint", id, v)
				return
			}
			(*n)[i] = uint(u64)
		}
	case *[]uint32:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]uint32, len(a))
		for i, v := range a {
			var u64 uint64
			if u64, ctx.bindError = strconv.ParseUint(v, 10, 32); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint32", id, v)
				return
			}
			(*n)[i] = uint32(u64)
		}
	case *[]uint16:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]uint16, len(a))
		for i, v := range a {
			var u64 uint64
			if u64, ctx.bindError = strconv.ParseUint(v, 10, 16); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint16", id, v)
				return
			}
			(*n)[i] = uint16(u64)
		}
	case *[]uint8:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]uint8, len(a))
		for i, v := range a {
			var u64 uint64
			if u64, ctx.bindError = strconv.ParseUint(v, 10, 8); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid uint8/byte", id, v)
				return
			}
			(*n)[i] = uint8(u64)
		}
	case *[]float64:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]float64, len(a))
		for i, v := range a {
			if (*n)[i], ctx.bindError = strconv.ParseFloat(v, 64); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid float64", id, v)
				return
			}
		}
	case *[]float32:
		var a []string
		if a, ctx.bindError = ctx.getQueryStringArray(id); ctx.bindError != nil {
			return
		}
		*n = make([]float32, len(a))
		for i, v := range a {
			var f64 float64
			if f64, ctx.bindError = strconv.ParseFloat(v, 32); ctx.bindError != nil {
				ctx.bindError = fmt.Errorf("id(%s)'s value(%s) is invalid float32", id, v)
				return
			}
			(*n)[i] = float32(f64)
		}
	default:
		ctx.bindError = fmt.Errorf("invalid value type(%s) for id(%s)", reflect.TypeOf(v).String(), id)
	}
}

func (ctx *baseContext) getQueryStringArray(id string) ([]string, error) {
	var ret []string
	v, ok := ctx.vars[id]
	if ok {
		ret = []string{v}
	}
	ret = append(ret, ctx.request.URL.Query()[id]...)
	return ret, nil
}

func (ctx *baseContext) getQueryString(id string) (string, error) {
	ret, ok := ctx.vars[id]
	if ok {
		return ret, nil
	}
	return ctx.request.URL.Query().Get(id), nil
}
