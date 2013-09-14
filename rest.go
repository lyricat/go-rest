/*
Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.
*/
package rest

import (
	"fmt"
	"github.com/ant0ine/go-urlrouter"
	"net/http"
	"reflect"
)

// Rest handle the http request and call to correspond handler.
type Rest struct {
	router urlrouter.Router
}

// New return a Rest.
func New() *Rest {
	return &Rest{}
}

// Add add a service to rest.
func (r *Rest) Add(v interface{}) error {
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}
	if vv.Kind() != reflect.Struct {
		return fmt.Errorf("invalid service")
	}
	vt := vv.Type()
	for i, n := 0, vv.NumField(); i < n; i++ {
		field := vt.Field(i)
		if first := field.Name[0]; !('A' <= first && first <= 'Z') {
			continue
		}
		service, ok := vv.Field(i).Interface().(RestService)
		if !ok {
			continue
		}
		routes, err := service.MakeHandlers(field.Tag, v)
		if err != nil {
			return err
		}
		for path, endpoint := range routes {
			r.router.Routes = append(r.router.Routes, urlrouter.Route{
				PathExp: path,
				Dest:    endpoint,
			})
		}
		if err := r.router.Start(); err != nil {
			return err
		}
	}
	return nil
}

// ServeHTTP serve the http request.
func (r *Rest) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, vars, err := r.router.FindRoute(req.URL.Path)
	if err != nil || route == nil {
		http.NotFound(w, req)
		return
	}
	endpoint, ok := route.Dest.(*EndPoint)
	if !ok {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	endpoint.Call(w, req, vars)
}
