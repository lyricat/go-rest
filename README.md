go-rest
=======

Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.

[![Build Status](https://travis-ci.org/googollee/go-rest.png?branch=master)](https://travis-ci.org/googollee/go-rest/)

Install
-------

	$ go get github.com/googollee/go-rest

Document
--------

http://godoc.org/github.com/googollee/go-rest

Summary
-------

Define a service struct like this:

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
	func (r RestExample) HandleCreateHello(arg HelloArg) {
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
	func (r RestExample) HandleHello() HelloArg {
		if r.Vars() == nil {
			r.Error(http.StatusNotFound, fmt.Errorf("%+v", r.Vars()))
			return HelloArg{}
		}
		to := r.Vars()["to"]
		post, ok := r.post[to]
		if !ok {
			r.Error(http.StatusNotFound, fmt.Errorf("can't find hello to %s", to))
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
		to := r.Vars()["to"]
		if to == "" {
			r.Error(http.StatusBadRequest, fmt.Errorf("need to"))
			return
		}
		r.WriteHeader(http.StatusOK)
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

The field tag of Service configure the parameters of processor, like method, path, or function which 
will process the request.

The path of processor can capture arguments, which will pass to process function by order in path. Arguments
type can be string or int, or any type which kind is string or int. 

The default name of handler is the name of field prefix with "Handle",
like Watch handelr correspond HandleWatch method.

Get the http.Handler from RestExample:

	handler, err := rest.New(&RestExample{
		post:  make(map[string]string),
		watch: make(map[string]chan string),
	})
	http.ListenAndServe("127.0.0.1:8080", handler)

Or use gorilla mux and work with other http handlers:

	// import "github.com/gorilla/mux"
	router := mux.NewRouter()
	handler, err := rest.New(&RestExample{
		post:  make(map[string]string),
		watch: make(map[string]chan string),
	})
	router.PathPrefix(handler.Prefix()).Handle(handler)

Performance
-----------

The performance test is in perf_test.go:

 - BenchmarkHttpServeFull: post a string and response a string, through http.ListenAndServe which include connection payload.

 - BenchmarkRestServe: post to a fake node, through Rest.ServeHTTP. It's the time of Rest.ServeHTTP, which mainly url routing and preparing context

 - BenchmarkRestGet: no post, and response a string, through Rest.ServeHTTP.

 - BenchmarkRestPost: post a string and no response, through Rest.ServeHTTP.

 - BenchmarkRestFull: post a string and response a string, through Rest.ServeHTTP.

 - BenchmarkPlainGet: no post and response a string, without go-rest framework. It use to compare BenchmarkRestGet.

 - BenchmarkPlainPost: post a string and no response, without go-rest framework. It use to compare BenchmarkRestPost.

 - BenchmarkPlainFull: post a string and response a string, without go-rest framework. It use to compare BenchmarkRestFull.

The result in mu mbp list below:

	$ go test -test.bench=Bench*
	PASS
	BenchmarkHttpServeFull	    5000	    522543 ns/op
	BenchmarkRestServe	  500000	      4884 ns/op
	BenchmarkRestGet	  500000	      7312 ns/op
	BenchmarkRestPost	  200000	      9214 ns/op
	BenchmarkRestFull	  200000	     10504 ns/op
	BenchmarkPlainGet	  200000	      8585 ns/op
	BenchmarkPlainPost	  200000	      8786 ns/op
	BenchmarkPlainFull	  200000	      8788 ns/op
