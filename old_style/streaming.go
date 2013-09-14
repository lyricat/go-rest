package rest

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"
)

/*
Stream  wrap the connection when using streaming.
*/
type Stream struct {
	ctx        *context
	conn       net.Conn
	end        string
	marshaller Marshaller
}

func newStream(ctx *context, conn net.Conn, end string) (*Stream, error) {
	marshaller, ok := getMarshaller(ctx.mime)
	if !ok {
		return nil, errors.New("can't find marshaller for" + ctx.mime)
	}
	return &Stream{
		ctx:        ctx,
		conn:       conn,
		end:        end,
		marshaller: marshaller,
	}, nil
}

// Write data i as a frame to the connection.
func (s *Stream) Write(i interface{}) error {
	err := s.marshaller.Marshal(s.ctx.responseWriter, s.ctx.name, i)
	if err != nil {
		return err
	}
	if len(s.end) > 0 {
		_, err = s.ctx.responseWriter.Write([]byte(s.end))
		if err != nil {
			return err
		}
	}
	return nil
}

// Check connection is still alive.
func (s *Stream) Ping() error {
	s.conn.SetReadDeadline(time.Now().Add(time.Second / 10))
	b := make([]byte, 1024)
	_, err := s.conn.Read(b)
	if connErr, ok := err.(net.Error); ok {
		if connErr.Timeout() {
			return nil
		}
	}
	return err
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
	pathFormatter
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
	ret := &streamingNode{
		findex: f.Index,
		name_:  name,
	}
	if ft.NumIn() > 3 || ft.NumIn() < 2 {
		return nil, nil, fmt.Errorf("streaming(%s) input parameters should be 1 or 2.", ft.Name())
	}
	if ft.In(1).String() != "rest.Stream" {
		return nil, nil, fmt.Errorf("streaming(%s) first input parameters should be rest.Stream", ft.Name())
	}
	if ft.NumIn() == 3 {
		ret.requestType = ft.In(2)
	}

	if ft.NumOut() > 0 {
		return nil, nil, fmt.Errorf("streaming(%s) return should no return.", ft.Name())
	}

	ret.end = tag.Get("end")
	p.pathFormatter = formatter

	return []handler{ret}, []pathFormatter{formatter}, nil
}
