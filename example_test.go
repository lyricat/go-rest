package rest_test

import (
	"github.com/googollee/go-rest"
	"net/http"
	"time"
)

type RestExample struct {
	rest.Service `prefix:"/prefix" mime:"application/json" charset:"utf-8"`

	CreateHello rest.Processor `method:"POST" path:"/hello"`
	GetHello    rest.Processor `method:"GET" path:"/hello/:to" func:"HandleHello"`
	Watch       rest.Streaming `method:"GET" path:"/hello/:to/streaming"`

	post  map[string]string
	watch map[string]chan string
}

type HelloArg struct {
	To   string `json:"to"`
	Post string `json:"post"`
}

// Post example:
// > curl "http://127.0.0.1:8080/prefix/hello" -d '{"to":"rest", "post":"rest is powerful"}'
//
// No response
func (r RestExample) HandleCreateHello(ctx rest.Context, arg HelloArg) {
	r.post[arg.To] = arg.Post
	c, ok := r.watch[arg.To]
	if ok {
		select {
		case c <- arg.Post:
		default:
		}
	}
}

// Get example:
// > curl "http://127.0.0.1:8080/prefix/hello/rest"
//
// Response:
//   {"to":"rest","post":"rest is powerful"}
func (r RestExample) HandleHello(ctx rest.Context) HelloArg {
	to := ctx.Vars()["to"]
	post, ok := r.post[to]
	if !ok {
		ctx.Error(resp, http.StatusNotFound, 2, "can't find hello to %s", to)
		return HelloArg{}
	}
	return HelloArg{
		To:   to,
		Post: post,
	}
}

// Streaming example:
// > curl "http://127.0.0.1:8080/prefix/hello/rest/streaming"
//
// It create a long-live connection and will receive post content "rest is powerful"
// when running post example.
func (r RestExample) HandleWatch(s rest.Stream) {
	to := s.Vars()["to"]
	if to == "" {
		s.Error(s, http.StatusBadRequest, 3, "need to")
		return
	}
	c := make(chan string)
	r.watch[to] = c
	for {
		post := <-c
		s.SetDeadline(time.Now().Add(time.Second))
		err := s.Write(post)
		if err != nil {
			close(c)
			delete(r.watch, to)
			return
		}
	}
}

func Example() {
	instance := &RestExample{
		post:  make(map[string]string),
		watch: make(map[string]chan string),
	}
	rest, err := rest.New(instance)
	if err != nil {
		panic(err)
	}

	http.ListenAndServe("127.0.0.1:8080", rest)
}
