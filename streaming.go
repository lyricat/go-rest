package rest

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type connection struct {
	conn       net.Conn
	bufrw      *bufio.ReadWriter
	marshaller Marshaller
}

type innerStreaming struct {
	formatter pathFormatter
	funcIndex int
	end       string
	timeout   time.Duration

	locker      sync.RWMutex
	connections map[string]map[string]chan interface{}
}

/*
Define the streaming.

Valid tag:

 - method: Define the method of http request.
 - path: Define the path of http request.
 - func: Define the get-identity function, which signature like func() string.
 - mime: Define the default mime of request's and response's body. It overwrite the service one.
 - end: Define the end of one data when streaming working.
 - timeout: Define the timeout to check connection.

To be implement:
 - charset: Define the default charset of request's and response's body. It overwrite the service one.
 - scope: Define required scope when process.
*/
type Streaming struct {
	*innerStreaming
}

// Feed the all streaming with identity. The data will marshal to string and followed by end
func (s Streaming) Feed(identity string, data interface{}) {
	s.locker.RLock()
	defer s.locker.RUnlock()

	conns, ok := s.connections[identity]
	if !ok {
		return
	}
	for _, c := range conns {
		c <- data
	}
}

// Generate the path of http request to processor. The args will fill in by url order.
func (s Streaming) Path(args ...interface{}) (string, error) {
	return s.formatter.path(args...), nil
}

// Dissconnect all connection with identity.
func (s Streaming) Disconnect(identity string) {
	s.locker.Lock()
	defer s.locker.Unlock()

	for _, c := range s.connections[identity] {
		close(c)
	}
}

func (s Streaming) init(streaming reflect.Value, formatter pathFormatter, f reflect.Method, tag reflect.StructTag) error {
	ft := f.Type
	if ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.String {
		return fmt.Errorf("streaming(%s) must return (string, error)", f.Name)
	}

	timeout := 1
	if t := tag.Get("timeout"); t != "" {
		var err error
		timeout, err = strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("streaming(%s) has invalid timeout %s", f.Name, t)
		}
	}

	streaming.Field(0).Set(reflect.ValueOf(&innerStreaming{
		formatter:   pathFormatter(formatter),
		funcIndex:   f.Index,
		end:         tag.Get("end"),
		timeout:     time.Duration(timeout) * time.Second,
		connections: make(map[string]map[string]chan interface{}),
	}))

	return nil
}

func (s Streaming) handle(instance reflect.Value, ctx *context, args []reflect.Value) {
	f := instance.Method(s.funcIndex)
	ret := f.Call(args)
	if ctx.isError {
		return
	}

	identity := ret[0].Interface().(string)

	hj, ok := ctx.responseWriter.(http.Hijacker)
	if !ok {
		http.Error(ctx.responseWriter, "webserver doesn't support streaming", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(ctx.responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	response := "HTTP/1.1 200 OK\r\n"
	ctx.responseWriter.Header().Set("Connection", "keep-alive")
	ctx.responseWriter.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", ctx.mime))

	_, err = bufrw.Write([]byte(response))
	if err != nil {
		return
	}
	err = ctx.responseWriter.Header().Write(bufrw)
	if err != nil {
		return
	}
	_, err = bufrw.Write([]byte("\r\n"))
	if err != nil {
		return
	}
	err = bufrw.Flush()
	if err != nil {
		return
	}

	c := make(chan interface{})

	s.locker.Lock()
	conns, ok := s.connections[identity]
	if !ok {
		conns = make(map[string]chan interface{})
	}
	if c, ok := conns[ctx.request.RemoteAddr]; ok {
		close(c)
	}
	conns[ctx.request.RemoteAddr] = c
	s.connections[identity] = conns
	s.locker.Unlock()

	defer func() {
		s.locker.Lock()
		defer s.locker.Unlock()
		conn.Close()
		func() {
			// c may have closed
			defer func() { recover() }()
			close(c)
		}()
		if conns, ok := s.connections[identity]; ok {
			delete(conns, ctx.request.RemoteAddr)
			if len(conns) == 0 {
				delete(s.connections, identity)
			} else {
				s.connections[identity] = conns
			}
		}
	}()

	for {
		select {
		case data, ok := <-c:
			if !ok {
				return
			}
			err := ctx.marshaller.Marshal(bufrw, data)
			if err != nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(s.timeout))
			_, err = bufrw.Write([]byte(s.end))
			if err != nil {
				return
			}
			err = bufrw.Flush()
			if err != nil {
				return
			}
		case <-time.After(s.timeout):
			buf := make([]byte, 1)
			conn.SetReadDeadline(time.Now().Add(time.Second / 1000))
			_, err := conn.Read(buf)
			if e, ok := err.(net.Error); err == nil || (ok && e.Timeout()) {
				continue
			}
			return
		}
	}
}
