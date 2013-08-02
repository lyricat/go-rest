package rest

import (
	"fmt"
	"github.com/ant0ine/go-urlrouter"
	"reflect"
	"strings"
)

type RoomSetter interface {
	SetRoom(ctx Context) error
}

type Handler struct {
	routes []urlrouter.Route
}

func New(raw interface{}) (*Handler, error) {
	v := reflect.ValueOf(raw)
	vi := reflect.Indirect(v)
	t := vi.Type()
	router := new(urlrouter.Router)
	router.Start()

	for i, n := 0, t.NumField(); i < n; i++ {
		field := t.Field(i)
		handlerValue := reflect.New(field.Type)
		if !handlerValue.Type().Implements(handlerType) {
			continue
		}
		h := handlerValue.Interface().(handler)

		funcName := strings.ToUpper(field.Name[:1]) + field.Name[1:]
		f := v.MethodByName(funcName)
		if !f.IsValid() {
			return nil, fmt.Errorf("can't find method %s for node %s", funcName, field.Name)
		}

		err := h.init(f)
		if err != nil {
			return nil, fmt.Errorf("init node %s error: %s", funcName, err)
		}

		path := normalizePath(field.Tag.Get("path"))

		dest, _, err := router.FindRoute(path)
		if err != nil {
			return nil, fmt.Errorf("invalid path tag \"%s\" of node %s", path, field.Name)
		}
		if dest == nil {
			router.Routes = append(router.Routes, urlrouter.Route{
				PathExp: path,
				Dest: &node{
					path: path,
					handlers: map[string]*namedNode{
						h.method(): &namedNode{
							name:    field.Name,
							handler: h,
						},
					},
				},
			})
			err := router.Start()
			if err != nil {
				return nil, fmt.Errorf("invalid path tag \"%s\" of node %s", path, field.Name)
			}
		} else {
			if dest.PathExp != path {
				return nil, fmt.Errorf("different key name of one path: %s at %s, %s", path, field.Name, dest.PathExp)
			}
			node := dest.Dest.(*node)
			if namedHandler, ok := node.handlers[h.method()]; ok {
				return nil, fmt.Errorf("method %s confict: %s, %s", h.method(), field.Name, namedHandler.name)
			}
			node.handlers[h.method()] = &namedNode{
				name:    field.Name,
				handler: h,
			}
		}
	}

	return &Handler{
		routes: router.Routes,
	}, nil
}
