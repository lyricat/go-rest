package rest

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

type pathFormatter string

func (f pathFormatter) path(args ...interface{}) string {
	return fmt.Sprintf(string(f), args...)
}

type nodeInterface interface {
	init(node reflect.Value, formatter pathFormatter, f reflect.Method, tag reflect.StructTag) error
	handle(instance reflect.Value, ctx *context, args []reflect.Value)
}

type node struct {
	nodeInterface

	path     *regexp.Regexp
	request  reflect.Type
	method   string
	argKinds []reflect.Kind
}

func newNode(t reflect.Type, prefix string, node_ reflect.Value, nodeType reflect.StructField) (*node, error) {
	method := nodeType.Tag.Get("method")
	if method == "" {
		return nil, fmt.Errorf("%s node's tag must contain method", nodeType.Name)
	}

	pathStr := nodeType.Tag.Get("path")
	if pathStr == "" {
		return nil, fmt.Errorf("%s node's tag must contain path", nodeType.Name)
	}
	path, err := parsePath(prefix, pathStr)
	if err != nil {
		return nil, err
	}

	funcName := nodeType.Tag.Get("func")
	if funcName == "" {
		funcName = nodeType.Name + "_"
	}
	f, ok := t.MethodByName(funcName)
	if !ok {
		return nil, fmt.Errorf("%s can't find method with name %s", t.Name(), funcName)
	}

	kinds, requestType, err := parseRequestType(path, f.Type)
	if err != nil {
		return nil, err
	}

	formatter := parsePathFormatter(path, kinds)

	err = node_.Interface().(nodeInterface).init(node_, formatter, f, nodeType.Tag)
	if err != nil {
		return nil, fmt.Errorf("%s %s", nodeType.Name, err)
	}

	return &node{
		nodeInterface: node_.Interface().(nodeInterface),
		path:          path,
		request:       requestType,
		method:        method,
		argKinds:      kinds,
	}, nil
}

func (n node) match(method, path string) ([]reflect.Value, error) {
	if n.method != method {
		return nil, fmt.Errorf("invalid method")
	}
	matchs := n.path.FindAllStringSubmatch(path, -1)
	if len(matchs) == 0 {
		return nil, fmt.Errorf("can't match")
	}
	pathArgs := matchs[0]
	var ret []reflect.Value
	for i, arg := range pathArgs[1:] {
		switch n.argKinds[i] {
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

func parsePath(root, path string) (*regexp.Regexp, error) {
	if root[len(root)-1] == '/' {
		root = root[:len(root)-1]
	}
	if path[0] != '/' {
		path = "/" + path
	}
	ret, err := regexp.Compile("^" + root + path + "$")
	if err != nil {
		return nil, fmt.Errorf("invalid path: %s", err)
	}
	return ret, nil
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

func parsePathFormatter(path *regexp.Regexp, kinds []reflect.Kind) pathFormatter {
	ret := path.String()
	if ret[0] == '^' {
		ret = ret[1:]
	}
	if ret[len(ret)-1] == '$' {
		ret = ret[:len(ret)-1]
	}
	argRegexp := regexp.MustCompile(`\(.*?\)`)
	for i, match := 0, argRegexp.FindStringSubmatchIndex(ret); match != nil; i, match = i+1, argRegexp.FindStringSubmatchIndex(ret) {
		start, end := match[0], match[1]
		endPart := ret[end:]
		ret = ret[:start]
		switch kinds[i] {
		case reflect.Int:
			ret += "%d"
		case reflect.String:
			ret += "%s"
		default:
			ret += "%v"
		}
		ret += endPart
	}
	return pathFormatter(ret)
}
