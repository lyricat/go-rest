package rest

import (
	"fmt"
	"net/http"
)

// EndPoint is a collect of method handlers in one url path.
type EndPoint struct {
	funcs   map[string]Handler
	methods string
}

// NewEndPoint create a EndPoint.
func NewEndPoint() *EndPoint {
	return &EndPoint{
		funcs: make(map[string]Handler),
	}
}

// Add add a handler of http method to endpoint.
func (p *EndPoint) Add(method string, handler Handler) error {
	if len(method) == 0 {
		return fmt.Errorf("method invalid")
	}
	if handler == nil {
		return fmt.Errorf("handler invalid")
	}

	if _, ok := p.funcs[method]; ok {
		return fmt.Errorf("%s method was set", method)
	}
	p.funcs[method] = handler
	if len(p.methods) == 0 {
		p.methods = method
	} else {
		p.methods += ", " + method
	}
	return nil
}

// Methods return all methods this endpoint processing.
func (p *EndPoint) Methods() string {
	return p.methods
}

// Call process http request r and response writer w, with url parameters vars.
// When calling, it will automatically using handler of method in request.
func (p *EndPoint) Call(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	if method := r.URL.Query().Get("_method"); method != "" {
		r.Method = method
	}
	if h, ok := p.funcs[r.Method]; ok {
		h.ServeHTTP(w, r, vars)
		return
	}
	w.Header().Set("Allow", p.methods)
	w.WriteHeader(http.StatusMethodNotAllowed)
}
