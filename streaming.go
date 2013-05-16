package rest

import (
	"fmt"
	"net"
	"reflect"
	"time"
)

/*
Stream  wrap the connection when using streaming.
*/
type Stream struct {
	*context
	conn net.Conn
	end  string
}

func newStream(ctx *context, conn net.Conn, end string) *Stream {
	return &Stream{
		context: ctx,
		conn:    conn,
		end:     end,
	}
}

// Write data i as a frame to the connection.
func (s *Stream) Write(i interface{}) error {
	err := s.marshaller.Marshal(s.context.responseWriter, i)
	if err != nil {
		return err
	}
	if len(s.end) > 0 {
		_, err = s.conn.Write([]byte(s.end))
		if err != nil {
			return err
		}
	}
	return nil
}

// SetDeadline sets the connection's network read & write deadlines.
func (s *Stream) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

// SetWriteDeadline sets the connection's network write deadlines.
func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}

/*
Define the streaming.

The streaming's handle function may take 1 or 2 input parameters and no return:

 - func Handler(s rest.Stream) or
 - func Handler(s rest.Stream, post PostType)

First parameter Stream is use for sending data when connecting.

Valid tag:

 - method: Define the method of http request.
 - path: Define the path of http request.
 - func: Define the get-identity function, which signature like func() string.
 - mime: Define the default mime of request's and response's body. It overwrite the service one.
 - end: Define the end of one data when streaming working.
*/
type Streaming struct {
	formatter pathFormatter
}

// Generate the path of url to processor. Map args fill parameters in path.
func (p Streaming) PathMap(args map[string]string) string {
	return p.formatter.pathMap(args)
}

// Generate the path of url to processor. It accepts a sequence of key/value pairs, and fill parameters in path.
func (p Streaming) Path(args ...string) string {
	return p.formatter.path(args...)
}

func (p *Streaming) init(formatter pathFormatter, instance reflect.Type, name string, tag reflect.StructTag) ([]handler, []pathFormatter, error) {
	fname := tag.Get("func")
	if fname == "" {
		fname = "Handle" + name
	}
	f, ok := instance.MethodByName(fname)
	if !ok {
		return nil, nil, fmt.Errorf("can't find handler: %s", fname)
	}

	ft := f.Type
	ret := new(streamingNode)
	ret.funcIndex = f.Index
	if ft.NumIn() > 3 || ft.NumIn() < 2 {
		return nil, nil, fmt.Errorf("streaming(%s) input parameters should be 1 or 2.", f.Name)
	}
	if ft.In(1).String() != "rest.Stream" {
		return nil, nil, fmt.Errorf("streaming(%s) first input parameters should be rest.Stream", f.Name)
	}
	if ft.NumIn() == 3 {
		ret.requestType = ft.In(2)
	}

	if ft.NumOut() > 0 {
		return nil, nil, fmt.Errorf("streaming(%s) return should no return.", f.Name)
	}

	ret.end = tag.Get("end")
	p.formatter = formatter

	return []handler{ret}, []pathFormatter{formatter}, nil
}
