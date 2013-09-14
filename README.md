go-rest
=======

Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.

[![Build Status](https://travis-ci.org/googollee/go-rest.png?branch=master)](https://travis-ci.org/googollee/go-rest/)[![Build Status](https://drone.io/github.com/googollee/go-rest/status.png)](https://drone.io/github.com/googollee/go-rest/latest)[![Coverage Status](https://coveralls.io/repos/googollee/go-rest/badge.png?branch=master)](https://coveralls.io/r/googollee/go-rest?branch=master)

Why another rest framework?
---------------------------

I want to make a clear, flexible framework, which easy to write and test. Here is some principle when I design go-rest:

 - One place configure.

 	All configure is set when defining service. No need to switch between service define and service launching to check runtime parameters.

 - Don't repeat yourself.

 	The framework will grab as many as it can to set up service. No need to specify "postdata" type or "output" type which has defined in handler.

 - Handle HTTP directly if you want.

 	Service.Request(), Service.Header() and Service.WriteHeader(int) can handle HTTP request or response directly. Make sure not to special post type if you want to handle request.Body directly.

 - Handle HTTP request right.

 	Use Accept-Encoding to check compression and use Accept to check mime. If Accept doesn't exist, use service default mime.

 - Work with other framework.

 	Package go-rest will work with GAE or net/http or any other framework working with http.Handler.

 - Speed.

 	Golang is fast, framework should be fast too. Performance benchmark is included in perf_test.go. you can compare go-rest and raw handler(with regexp to route) performance.

 - Easy to do unit test.

 	No need to worry about marshal and unmarshal when do unit test, test handle function with input or output arguments directly. (using rest.SetTest)

Install
-------

	$ go get github.com/googollee/go-rest

Document
--------

http://godoc.org/github.com/googollee/go-rest

A over all example
-------

Define a service struct like this:

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

The field tag of Service configure the parameters of node, like method or path which will process the request.
