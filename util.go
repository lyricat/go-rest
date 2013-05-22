package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
)

// Test a service with special vars and request. If tested handler doesn't access vars or request, set them to nil.
func SetTest(i interface{}, vars map[string]string, r *http.Request) (*httptest.ResponseRecorder, error) {
	instance := reflect.ValueOf(i)
	var service reflect.Value
	index := 0
	for i, n := 0, instance.NumField(); i < n; i++ {
		field := instance.Field(i)
		if field.Type().String() == "rest.Service" {
			service, index = field, i
		}
	}
	if !service.IsValid() {
		return nil, fmt.Errorf("%s doesn't contain rest.Service field.", instance.Type().Name())
	}
	_, mime, charset, err := initService(service, instance.Type().Field(index).Tag)
	if err != nil {
		return nil, err
	}
	w := httptest.NewRecorder()
	ctx, err := newContext(w, r, vars, mime, charset)
	if err != nil {
		return nil, err
	}
	ctxField := service.FieldByName("context")
	ctxField.Set(reflect.ValueOf(ctx))
	return w, nil
}
