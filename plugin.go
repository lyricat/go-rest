package rest

import (
	"net/http"
)

// Response hold the field for web response, which use in plugin.
type Response struct {
	// Response code, like 200 or 404.
	Status int

	// Response header.
	Header http.Header

	// Response body
	Body string
}

// Plugin is a set of function which will be called at special stage when handling a request.
type Plugin interface {
	// PreProcessor will be called just before a Processor function run.
	// PreProcessor can change the request passing to function, e.g. parse the auth data and set scope to header.
	// If PreProcessor's return is not nil, it will ignore the processor and return immediately.
	PreProcessor(request *http.Request, service Service, processor Processor) *Response

	// Response will be called before returning http response.
	// Response can change the response which returned by handler function. The response will be returned as http response.
	Response(response *Response, service Service, processor Processor)
}
