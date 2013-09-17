package rest

import (
	"net/http"
	"net/http/httptest"
	"time"
)

// RecordContext is a implementation of Context that records its mutations for later inspection in tests.
type RecordContext struct {
	Context

	Req      *http.Request
	Recorder *httptest.ResponseRecorder
	Renders  []interface{}
}

// NewRecordContext create a RecordContext with a http request and url's parameters.
func NewRecordContext(vars map[string]string, req *http.Request) *RecordContext {
	resp := httptest.NewRecorder()
	return &RecordContext{
		Context: newBaseContext("test", nil, "utf-8", vars, req, resp),

		Req:      req,
		Recorder: resp,
	}
}

// Render implement Context's Render.
func (ctx *RecordContext) Render(v interface{}) error {
	ctx.Renders = append(ctx.Renders, v)
	return ctx.Context.Render(v)
}

// Ping implement StreamContext's Ping.
func (ctx *RecordContext) Ping() error {
	return nil
}

// SetWriteDeadline implement StreamContext's SetWriteDeadline.
func (ctx *RecordContext) SetWriteDeadline(t time.Time) error {
	return nil
}
