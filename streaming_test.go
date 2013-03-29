package rest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

type TestStreaming struct {
	Service `prefix:"/test" mime:"application/json" charset:"utf-8"`

	Connection Streaming `method:"GET" path:"/connection" func:"Connection_" end:""`
	Error      Streaming `method:"GET" path:"/error"`
}

func (t TestStreaming) Connection_() string {
	return t.Request().URL.Query().Get("token")
}

func (t TestStreaming) Error_() string {
	t.Service.Error(http.StatusBadGateway, fmt.Errorf("error"))
	return ""
}

func TestStreamingService(t *testing.T) {
	s := new(TestStreaming)
	handler, err := New(s)
	if err != nil {
		t.Fatalf("create error: %s", err)
	}
	go http.ListenAndServe(":28888", handler)

	{
		c := make(chan int)
		go func() {
			<-c
			s.Connection.Feed("123", 1)
			<-c
			s.Connection.Feed("123", 2)
			s.Connection.Feed("123", "abc")
			c <- 1
			c <- 1
			s.Connection.Disconnect("123")
			c <- 1
		}()

		resp1, err := http.Get("http://localhost:28888/test/connection?token=123")
		if err != nil {
			t.Fatal(err)
		}
		defer resp1.Body.Close()
		if resp1.StatusCode != http.StatusOK {
			b, _ := ioutil.ReadAll(resp1.Body)
			t.Fatal(string(b))
		}
		c <- 1
		resp2, err := http.Get("http://localhost:28888/test/connection?token=123")
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			b, _ := ioutil.ReadAll(resp2.Body)
			t.Fatal(string(b))
		}
		c <- 1

		<-c

		expect1 := "1\n2\n\"abc\"\n"
		buf1 := make([]byte, len(expect1))
		n, err := resp1.Body.Read(buf1)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf1[:n]) != expect1 {
			t.Errorf("not expect: %s", string(buf1[:n]))
		}

		expect2 := "2\n\"abc\"\n"
		buf2 := make([]byte, len(expect2))
		n, err = resp2.Body.Read(buf2)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf2[:n]) != expect2 {
			t.Errorf("not expect: %s", string(buf2[:n]))
		}

		<-c
		<-c
		_, err = resp1.Body.Read(buf1)
		if err == nil {
			t.Errorf("resp1 should be closed")
		}
		_, err = resp2.Body.Read(buf2)
		if err == nil {
			t.Errorf("resp2 should be closed")
		}
	}

	{
		resp, err := http.Get("http://127.0.0.1:28888/test/error")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadGateway {
			t.Errorf("not expect code: %s", resp.Status)
		}
	}
}
