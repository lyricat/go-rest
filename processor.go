package rest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

type innerProcessor struct {
	method       string
	path         *regexp.Regexp
	pathArgKinds []reflect.Kind
	requestType  reflect.Type
	responseType reflect.Type
	realm        []string
	funcIndex    int
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
	if len(args) != len(p.pathArgKinds) {
		return "", fmt.Errorf("arguments' number %d not meet requist %d", len(args), len(p.pathArgKinds))
	}
	path := p.path.String()
	if path[len(path)-1] == '$' {
		path = path[:len(path)-1]
	}
	argRegexp := regexp.MustCompile(`\(.*?\)`)
	for i, match := 0, argRegexp.FindStringSubmatchIndex(path); match != nil; i, match = i+1, argRegexp.FindStringSubmatchIndex(path) {
		start, end := match[0], match[1]
		path = path[:start] + fmt.Sprintf("%v", args[i]) + path[end:]
	}
	return path, nil
}

func (p Processor) getArgs(path string) ([]reflect.Value, error) {
	pathArgs := p.path.FindAllStringSubmatch(path, -1)[0]
	var ret []reflect.Value
	for i, arg := range pathArgs[1:] {
		switch p.pathArgKinds[i] {
		case reflect.Int:
			i, err := strconv.Atoi(arg)
			if err != nil {
				return nil, fmt.Errorf("can't convert %s of path %s to int: %s", arg, path, err)
			}
			ret = append(ret, reflect.ValueOf(i))
		default:
			ret = append(ret, reflect.ValueOf(arg))
		}
	}
	return ret, nil
}

func initProcessor(root string, processor reflect.Value, tag reflect.StructTag, f reflect.Method) error {
	path, err := parsePath(root, tag.Get("path"))
	if err != nil {
		return err
	}
	kinds, requestType, err := parseRequestType(path, f.Type)
	if err != nil {
		return err
	}
	retType, err := parseResponseType(f.Type)
	if err != nil {
		return err
	}

	processor.Field(0).Set(reflect.ValueOf(&innerProcessor{
		method:       tag.Get("method"),
		path:         path,
		pathArgKinds: kinds,
		requestType:  requestType,
		responseType: retType,
		realm:        parseRealm(tag),
		funcIndex:    f.Index,
	}))

	return nil
}

func parsePath(root, path string) (*regexp.Regexp, error) {
	if root[len(root)-1] == '/' {
		root = root[:len(root)-1]
	}
	if path[0] != '/' {
		path = "/" + path
	}
	ret, err := regexp.Compile(root + path + "$")
	if err != nil {
		return nil, fmt.Errorf("invalid path: %s", err)
	}
	return ret, nil
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

func parseRequestType(path *regexp.Regexp, f reflect.Type) (kinds []reflect.Kind, post reflect.Type, err error) {
	pathArgLen := len(path.SubexpNames()) - 1
	funcArgLen := f.NumIn() - 1
	if funcArgLen < pathArgLen || funcArgLen > pathArgLen+1 {
		err = fmt.Errorf("url(%s) arguments number %d can't match processor(%s) arguments number %d", path, pathArgLen, f.Name(), funcArgLen)
		return
	}
	for i, n := 0, pathArgLen; i < n; i++ {
		kind := f.In(i + 1).Kind()
		if i < pathArgLen {
			if kind != reflect.String && kind != reflect.Int {
				switch i {
				case 0:
					err = fmt.Errorf("processor(%s) 1st argument's kind %s is not valid, must string or int", f.Name(), kind)
				case 1:
					err = fmt.Errorf("processor(%s) 2nd argument's kind %s is not valid, must string or int", f.Name(), kind)
				case 2:
					err = fmt.Errorf("processor(%s) 3rd argument's kind %s is not valid, must string or int", f.Name(), kind)
				default:
					err = fmt.Errorf("processor(%s) %dth argument's kind %s is not valid, must string or int", f.Name(), i+1, kind)
				}
				return
			}
		}
		kinds = append(kinds, kind)
	}
	if funcArgLen == pathArgLen+1 {
		post = f.In(f.NumIn() - 1)
		if !checkTypeCanMarshal(post) {
			err = fmt.Errorf("processor(%s)'s return type %s can't be marshaled", f.Name(), post.String())
		}
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
