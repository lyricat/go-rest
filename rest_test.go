package rest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

type FullTest struct {
	Service `root:"/test/" realm:"tester"`

	Hello  Processor `method:"GET" path:"/hello/([a-zA-Z0-9]+)"`
	Print  Processor `method:"POST" path:"/print/([0-9]+)"`
	Error_ Processor `method:"GET" path:"/error" func:"ErrorFunc"`
}

func (t FullTest) Hello_(guest string) string {
	path, _ := t.Hello.Path(guest)
	t.RedirectTo(path)
	return "hello " + guest
}

func (t FullTest) Print_(id int, post string) string {
	ret := fmt.Sprintf("id(%d) post: %s", id, post)
	t.Header().Set("Type", "abcd")
	path, _ := t.Hello.Path("guest")
	t.Header().Set("Location", path)
	t.Response(http.StatusCreated)
	return ret
}

func (t FullTest) ErrorFunc() string {
	t.Error(http.StatusInternalServerError, fmt.Errorf("error: %s", "no reason"))
	return ""
}

func TestRestful(t *testing.T) {
	test := new(FullTest)
	handler, err := Init(test)
	if err != nil {
		t.Fatalf("can't init test: %s", err)
	}
	go func() {
		err := http.ListenAndServe(":12345", handler)
		if err != nil {
			panic(err)
		}
	}()

	{
		r, err := http.NewRequest("GET", "http://127.0.0.1:12345/test/hello/restful", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, status, header := sendRequest(handler, r)
		if status != http.StatusTemporaryRedirect {
			t.Errorf("call hello/restful status not redirect: %d", status)
		}
		if header.Get("Location") != "/test/hello/restful" {
			t.Errorf("call hello/restful location error: %s", header.Get("Location"))
		}
		if resp != "\"hello restful\"\n" {
			t.Errorf("call hello/restful response error: [%s]", resp)
		}
	}

	{
		buf := bytes.NewBufferString(`"post content"`)
		r, err := http.NewRequest("POST", "http://127.0.0.1:12345/test/print/123", buf)
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, status, header := sendRequest(handler, r)
		if status != http.StatusCreated {
			t.Errorf("call print/123 status not created: %d", status)
		}
		if header.Get("Type") != "abcd" {
			t.Errorf("call print/123 Type error: %s", header.Get("Typa"))
		}
		if header.Get("Location") != "/test/hello/guest" {
			t.Errorf("call print/123 location error: %s", header.Get("Location"))
		}
		if resp != "\"id(123) post: post content\"\n" {
			t.Errorf("call print/123 response error: [%s]", resp)
		}
	}

	{
		r, err := http.NewRequest("GET", "http://127.0.0.1:12345/test/error", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, status, _ := sendRequest(handler, r)
		if status != http.StatusInternalServerError {
			t.Errorf("call error status not redirect: %s", status)
		}
		if resp != "error: no reason" {
			t.Errorf("call error response error: [%s]", resp)
		}
	}
}

type NoServiceTest struct{}
type ServiceNotFirstTest struct {
	a int
	Service
}
type NoHandlerService struct {
	Service
	Hello Processor
}
type HandlerNotMatchService struct {
	Service
	Hello Processor `path:"/hello/(.*?)"`
}

func (s HandlerNotMatchService) Hello_() {}

func TestServiceError(t *testing.T) {
	var tests = []interface{}{
		1,
		new(NoServiceTest),
		new(ServiceNotFirstTest),
		new(NoHandlerService),
		new(HandlerNotMatchService),
	}

	for i, test := range tests {
		_, err := Init(test)
		if err == nil {
			t.Errorf("test %d should error", i)
		}
	}
}

func respHelper(resp *http.Response, e error) (ret string, code int, header http.Header, err error) {
	if e != nil {
		err = e
		return
	}
	defer resp.Body.Close()
	code = resp.StatusCode
	header = resp.Header
	body, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		panic(e)
	}
	ret = string(body)
	return
}

type responseWriter struct {
	status int
	header http.Header
	buf    *bytes.Buffer
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		status: http.StatusOK,
		header: make(http.Header),
		buf:    bytes.NewBuffer(nil),
	}
}

func (w *responseWriter) Header() http.Header {
	return w.header
}

func (w *responseWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
}

func sendRequest(handler http.Handler, r *http.Request) (ret string, code int, header http.Header) {
	resp := newResponseWriter()
	handler.ServeHTTP(resp, r)
	return resp.buf.String(), resp.status, resp.header
}

type RESTService struct {
	Service `root:"/root"`

	Hello    Processor `path:"/hello/(.*?)/to/(.*?)" method:"GET"`
	PostConv Processor `path:"/conversation" func:"PostConversation" method:"POST"`
	Conv     Processor `path:"/conversation/([0-9]+)" func:"GetConversation" method:"GET"`
}

func (s RESTService) Hello_(host, guest string) string {
	return "hello from " + host + " to " + guest
}

func (s RESTService) PostConversation(post string) string {
	path, _ := s.Conv.Path(1)
	s.RedirectTo(path)
	return "just post: " + post
}

func (s RESTService) GetConversation(id int) string {
	return fmt.Sprintf("get post id %d", id)
}

func ExampleRest() {
	handler, _ := Init(new(RESTService))
	http.ListenAndServe("127.0.0.1:8080", handler)
}
