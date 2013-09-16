package rest

import (
	"fmt"
)

// CheckRoute check which handler will handle the request with path and method in rest r.
func CheckRoute(r *Rest, path, method string) (string, map[string]string, error) {
	route, vars, err := r.router.FindRoute(path)
	if err != nil {
		return "", nil, err
	}
	if route == nil {
		return "", nil, fmt.Errorf("can't find path %s handelr", path)
	}
	endpoint, ok := route.Dest.(*EndPoint)
	if !ok {
		return "", nil, fmt.Errorf("path %s handler is invalid", path)
	}
	handler, ok := endpoint.funcs[method]
	if !ok {
		return "", nil, fmt.Errorf("path %s can't handle method %s", path, method)
	}
	return handler.Name(), vars, nil
}
