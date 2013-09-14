package rest

import (
	"fmt"
	"reflect"
)

// RestService is service creator.
type RestService interface {
	// MakeHandlers use service tag and service instance v to create a set of endpoint.
	// Service is a implementation of RestService.
	MakeHandlers(tag reflect.StructTag, v interface{}) (map[string]*EndPoint, error)
}

// Service is a rest service creator.
type Service struct{}

// MakeHandlers will use v's nodes to create a set of endpoint.
func (s Service) MakeHandlers(tag reflect.StructTag, v interface{}) (map[string]*EndPoint, error) {
	sv := reflect.ValueOf(v)
	st := sv.Type()
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}
	ret := make(map[string]*EndPoint)
	for i, n := 0, st.NumField(); i < n; i++ {
		field := st.Field(i)
		node, ok := reflect.New(field.Type).Elem().Interface().(Node)
		if !ok {
			continue
		}
		fname := upperFirst(field.Name)
		f := sv.MethodByName(fname)
		if !f.IsValid() {
			return nil, fmt.Errorf("can't find %s node handler: %s", field.Name, fname)
		}
		path, method, handler, err := node.CreateHandler(tag, field.Tag, fname, f)
		if err != nil {
			return nil, err
		}
		endpoint, ok := ret[path]
		if !ok {
			endpoint = NewEndPoint()
			ret[path] = endpoint
		}
		if err := endpoint.Add(method, handler); err != nil {
			return nil, err
		}
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}
