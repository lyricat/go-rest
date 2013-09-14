package rest_test

import (
	"net/http"
	"time"

	"github.com/googollee/go-rest"
	"github.com/googollee/go-rest/pubsub"
)

type RestExample struct {
	rest.Service `prefix:"/prefix" mime:"application/json"`

	createHello rest.SimpleNode `method:"POST" route:"/hello"`
	getHello    rest.SimpleNode `method:"GET" route:"/hello/:to"`
	watch       rest.Streaming  `method:"GET" path:"/hello/:to/streaming"`

	post   map[string]string
	pubsub *pubsub.Pubsub
}

type HelloArg struct {
	To   string `json:"to"`
	Post string `json:"post"`
}

// Post example:
// > curl "http://127.0.0.1:8080/prefix/hello" -d '{"to":"rest", "post":"rest is powerful"}'
//
// No response
func (r *RestExample) CreateHello(ctx rest.Context, arg HelloArg) {
	r.post[arg.To] = arg.Post
	r.pubsub.Publish(arg.To, arg.Post)
}

// Get example:
// > curl "http://127.0.0.1:8080/prefix/hello/rest"
//
// Response:
//   {"to":"rest","post":"rest is powerful"}
func (r *RestExample) Hello(ctx rest.Context) {
	var to string
	ctx.Bind("to", &to)
	post, ok := r.post[to]
	if !ok {
		ctx.Return(http.StatusNotFound, "can't find hello to %s", to)
		return
	}
	ctx.Render(HelloArg{
		To:   to,
		Post: post,
	})
}

// Streaming example:
// > curl "http://127.0.0.1:8080/hello/rest/streaming"
//
// It create a long-live connection and will receive post content "rest is powerful"
// when running post example.
func (r *RestExample) Watch(ctx rest.StreamContext) {
	var to string
	ctx.Bind("to", &to)
	if to == "" {
		ctx.Return(http.StatusBadRequest, "invalid to")
		return
	}
	ctx.Return(http.StatusOK)

	c := make(chan interface{})
	r.pubsub.Subscribe(to, c)
	defer r.pubsub.UnsubscribeAll(c)

	for ctx.Ping() == nil {
		select {
		case post := <-c:
			ctx.SetWriteDeadline(time.Now().Add(time.Second))
			if err := ctx.Render(post); err != nil {
				return
			}
		case <-time.After(time.Second):
		}
	}
}

// The usage of rest.
func Example() {
	instance := &RestExample{
		post:   make(map[string]string),
		pubsub: pubsub.New(3),
	}
	r := rest.New()
	if err := r.Add(instance); err != nil {
		panic(err)
	}

	http.ListenAndServe("127.0.0.1:8080", r)
}
