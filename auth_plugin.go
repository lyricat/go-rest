package rest

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type AuthPlugin struct {
	getScope func(r *http.Request) []string
}

func NewAuthPlugin(f func(r *http.Request) []string) *AuthPlugin {
	return &AuthPlugin{
		getScope: f,
	}
}

func (a *AuthPlugin) PreProcess(r *http.Request, s Service, p Processor) *Response {
	scopes := a.getScope(r)

	ok, lack := a.checkScope(scopes, s.Tag)
	if !ok {
		return &Response{
			Status: http.StatusUnauthorized,
			Body:   fmt.Sprintf("request lack scope %s", lack),
		}
	}

	ok, lack = a.checkScope(scopes, p.Tag)
	if !ok {
		return &Response{
			Status: http.StatusUnauthorized,
			Body:   fmt.Sprintf("request lack scope %s", lack),
		}
	}
	return nil
}

func (a *AuthPlugin) Response(r *Response, s Service, p Processor) {}

func (a *AuthPlugin) checkScope(userScope []string, tag reflect.StructTag) (bool, string) {
	scope := tag.Get("scope")
	if scope == "*" {
		return true, ""
	}
	match := make(map[string]bool)
	for _, s := range strings.Split(scope, ",") {
		match[s] = false
	}
	for _, s := range userScope {
		match[s] = true
	}
	for k, v := range match {
		if !v {
			return false, k
		}
	}
	return true, ""
}
