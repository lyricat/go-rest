package rest

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var invalidHandler = errors.New("invalid handler")

type pathFormatter string

func pathToFormatter(prefix, path string) pathFormatter {
	if len(prefix) == 0 || prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if l := len(prefix); prefix[l-1] == '/' {
		prefix = prefix[:l-1]
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return pathFormatter(prefix + path)
}

func (f pathFormatter) pathMap(args map[string]string) string {
	ret := string(f)
	for k, v := range args {
		ret = strings.Replace(ret, ":"+k, v, -1)
	}
	return ret
}

func (f pathFormatter) path(params ...string) string {
	var key string
	m := make(map[string]string)
	for i, p := range params {
		if i&1 == 0 {
			key = p
		} else {
			m[key] = p
			key = ""
		}
	}
	if key != "" {
		m[key] = ""
	}
	return f.pathMap(m)
}

type handlerInterface interface {
	init(formatter pathFormatter, f reflect.Method, tag reflect.StructTag) error
	handle(instance reflect.Value, ctx *context)
}

type node struct {
	handler handlerInterface
	method  string
}

func newNode(instanceType reflect.Type, handler reflect.Value, prefix string, nodeType reflect.StructField) (*node, pathFormatter, error) {
	if handler.Kind() != reflect.Struct {
		return nil, "", invalidHandler
	}

	innerValue := handler.Field(0)
	if innerValue.Kind() != reflect.Ptr {
		return nil, "", invalidHandler
	}

	_, ok := handler.Field(0).Interface().(handlerInterface)
	if !ok {
		return nil, "", invalidHandler
	}
	innerType := innerValue.Type().Elem()
	handler.Field(0).Set(reflect.New(innerType))
	h := handler.Field(0).Interface().(handlerInterface)

	method := nodeType.Tag.Get("method")
	if method == "" {
		return nil, "", fmt.Errorf("%s node's tag must contain method", nodeType.Name)
	}

	pathStr := nodeType.Tag.Get("path")
	if pathStr == "" {
		return nil, "", fmt.Errorf("%s node's tag must contain path", nodeType.Name)
	}
	formatter := pathToFormatter(prefix, pathStr)

	funcName := nodeType.Tag.Get("func")
	if funcName == "" {
		funcName = "Handle" + nodeType.Name
	}
	f, ok := instanceType.MethodByName(funcName)
	if !ok {
		return nil, "", fmt.Errorf("%s can't find method with name %s", instanceType.Name(), funcName)
	}

	err := h.init(formatter, f, nodeType.Tag)
	if err != nil {
		return nil, "", fmt.Errorf("can't init node %s: %s", nodeType.Name, err)
	}

	return &node{
		handler: h,
		method:  method,
	}, formatter, nil
}
