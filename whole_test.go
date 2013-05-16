package rest

import (
	"bytes"
	"encoding/json"
	"github.com/stretchrcom/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type RestExample struct {
	Service `prefix:"/prefix" mime:"application/json" charset:"utf-8"`

	CreateHello Processor `method:"POST" path:"/hello"`
	GetHello    Processor `method:"GET" path:"/hello/:to" func:"HandleHello"`
	Watch       Streaming `method:"GET" path:"/hello/:to/streaming"`

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
func (r RestExample) HandleCreateHello(ctx Context, arg HelloArg) {
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
func (r RestExample) HandleHello(ctx Context) HelloArg {
	to := ctx.Vars()["to"]
	post, ok := r.post[to]
	if !ok {
		ctx.Error(http.StatusNotFound, 2, "can't find hello to %s", to)
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
func (r RestExample) HandleWatch(s Stream) {
	to := s.Vars()["to"]
	if to == "" {
		s.Error(http.StatusBadRequest, 3, "need 'to' parameter.")
		return
	}
	s.WriteHeader(http.StatusOK)
	c := make(chan string)
	r.watch[to] = c
	for {
		var post interface{}
		select {
		case <-time.After(time.Second):
			return
		case post = <-c:
		}
		s.SetDeadline(time.Now().Add(time.Second))
		err := s.Write(post)
		if err != nil {
			close(c)
			delete(r.watch, to)
			return
		}
	}
}

func TestError(t *testing.T) {
	type Test struct {
		url     string
		method  string
		request string

		code     int
		headers  http.Header
		response string
	}
	var tests = []Test{
		{"http://domain/prefix/nonexist", "GET", ``, http.StatusNotFound, http.Header{}, ""},
		{"http://domain/prefix/hello", "GET", ``, http.StatusNotFound, http.Header{}, ""},
		{"http://domain/prefix/hello", "POST", ``, http.StatusBadRequest, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, "{\"code\":-1,\"message\":\"marshal request to HelloArg failed: EOF\"}\n"},
		{"http://domain/prefix/hello", "POST", `{"to":"rest", "post":"rest is powerful"}`, http.StatusOK, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, ""},

		{"http://domain/prefix/hello/abc", "GET", ``, http.StatusNotFound, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, "{\"code\":2,\"message\":\"can't find hello to abc\"}\n"},
		{"http://domain/prefix/hello/rest", "GET", ``, http.StatusOK, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, "{\"to\":\"rest\",\"post\":\"rest is powerful\"}\n"},

		{"http://domain/prefix/hello/abc/streaming", "GET", ``, http.StatusInternalServerError, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, "{\"code\":-1,\"message\":\"webserver doesn't support hijacking\"}\n"},
	}
	r, err := New(&RestExample{
		post:  make(map[string]string),
		watch: make(map[string]chan string),
	})
	if err != nil {
		t.Fatalf("new rest service failed: %s", err)
	}
	assert.Equal(t, r.Prefix(), "/prefix")
	for i, test := range tests {
		buf := bytes.NewBufferString(test.request)
		req, err := http.NewRequest(test.method, test.url, buf)
		if err != nil {
			t.Fatalf("can't create request of test %d: %s", i, err)
		}
		resp := httptest.NewRecorder()
		resp.Code = http.StatusOK
		r.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, test.code, "test %d", i)
		assert.Equal(t, resp.Body.String(), test.response, "test %d", i)
		assert.Equal(t, resp.HeaderMap, test.headers, "test %d", i)
	}
}

func TestExample(t *testing.T) {
	instance := &RestExample{
		post:  make(map[string]string),
		watch: make(map[string]chan string),
	}
	rest, err := New(instance)
	if err != nil {
		t.Fatalf("create rest failed: %s", err)
	}

	assert.Equal(t, rest.Prefix(), "/prefix")

	server := httptest.NewServer(rest)
	defer func() {
		server.Close()
	}()

	resp, err := http.Get(server.URL + "/prefix/hello/rest")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)

	c := make(chan int)
	go func() {
		resp, err := http.Get(server.URL + "/prefix/hello//streaming")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
		assert.Equal(t, resp.Header, http.Header{"Connection": []string{"keep-alive"}, "Content-Type": []string{"application/json; charset=utf-8"}})
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, string(body), "{\"code\":3,\"message\":\"need 'to' parameter.\"}\n")

		resp, err = http.Get(server.URL + "/prefix/hello/rest/streaming")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, resp.StatusCode, http.StatusOK)

		expect := "\"rest is powerful\"\n"
		get := make([]byte, len(expect))
		n, err := resp.Body.Read(get)
		if err != nil {
			t.Fatal(err)
		}
		get = get[:n]
		assert.Equal(t, string(get), expect)

		c <- 1
	}()

	time.Sleep(time.Second / 2) // waiting streaming connected

	arg := HelloArg{
		To:   "rest",
		Post: "rest is powerful",
	}
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	err = encoder.Encode(arg)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(server.URL+"/prefix/hello", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	select {
	case <-c:
	case <-time.After(time.Second):
		t.Errorf("waiting streaming too long")
	}

	resp, err = http.Get(server.URL + "/prefix/hello/rest")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&arg)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, arg.To, "rest")
	assert.Equal(t, arg.Post, "rest is powerful")
}

type CompressExample struct {
	Service `compress:"on"`

	P Processor `path:"/p" method:"POST"`
	S Streaming `path:"/s" method:"GET"`
}

func (c CompressExample) HandleP(ctx Context) string {
	return "Hello"
}

func (c CompressExample) HandleS(s Stream) {
	s.Write("Hello")
}

func TestCompress(t *testing.T) {
	rest, err := New(new(CompressExample))
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(rest)
	defer server.Close()

	req, err := http.NewRequest("POST", server.URL+"/p", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, resp.Header.Get("Content-Encoding"), "gzip")
	assert.Equal(t, string(b), "\x1f\x8b\b\x00\x00\tn\x88\x00\xffR\xf2H\xcd\xc9\xc9W\xe2\x02\x04\x00\x00\xff\xffa\xeer\xd8\b\x00\x00\x00")

	req, err = http.NewRequest("GET", server.URL+"/s", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, resp.Header.Get("Content-Encoding"), "gzip")
	assert.Equal(t, string(b), "\x1f\x8b\b\x00\x00\tn\x88\x00\xffR\xf2H\xcd\xc9\xc9W\xe2\x02\x04\x00\x00\xff\xffa\xeer\xd8\b\x00\x00\x00")
}
