package rest

import (
	"fmt"
	"reflect"
	"testing"
)

type FakeStreaming struct {
	last map[string]string
}

func (f FakeStreaming) NoInput(s Stream) {
	f.last["method"] = "NoInput"
	f.last["input"] = ""
}

func (f FakeStreaming) Input(s Stream, input string) {
	f.last["method"] = "Input"
	f.last["input"] = input
}

func (f FakeStreaming) HandleNormal(s Stream) {
	f.last["method"] = "HandleNormal"
	f.last["input"] = ""
}

func (f FakeStreaming) ErrorEmpty() {}

func (f FakeStreaming) ErrorStream(input string) {}

func (f FakeStreaming) ErrorMore(s Stream, input string, other int) {}

func (f FakeStreaming) ErrorReturn(s Stream) string { return "" }

func TestStreamingInit(t *testing.T) {
	type Test struct {
		path pathFormatter
		name string
		tag  reflect.StructTag

		ok        bool
		funcIndex int
		request   string
		end       string
	}
	s := new(FakeStreaming)
	instance := reflect.ValueOf(s).Elem()
	instanceType := instance.Type()
	ni, ok := instanceType.MethodByName("NoInput")
	if !ok {
		t.Fatal("no NoInput")
	}
	i, ok := instanceType.MethodByName("Input")
	if !ok {
		t.Fatal("no Input")
	}
	hn, ok := instanceType.MethodByName("HandleNormal")
	if !ok {
		t.Fatal("no HandleNormal")
	}
	ee, ok := instanceType.MethodByName("ErrorEmpty")
	if !ok {
		t.Fatal("no ErrorEmpty")
	}
	es, ok := instanceType.MethodByName("ErrorStream")
	if !ok {
		t.Fatal("no ErrorStream")
	}
	em, ok := instanceType.MethodByName("ErrorMore")
	if !ok {
		t.Fatal("no ErrorMore")
	}
	er, ok := instanceType.MethodByName("ErrorReturn")
	if !ok {
		t.Fatal("no ErrorReturn")
	}
	var tests = []Test{
		{"/", "", `end:"\n" func:"NoInput"`, true, ni.Index, "<nil>", "\n"},
		{"/", "", `func:"Input"`, true, i.Index, "string", ""},
		{"/", "Normal", ``, true, hn.Index, "<nil>", ""},
		{"/", "", `func:"ErrorEmpty"`, false, ee.Index, "", ""},
		{"/", "", `func:"ErrorStream"`, false, es.Index, "", ""},
		{"/", "", `func:"ErrorMore"`, false, em.Index, "", ""},
		{"/", "", `func:"ErrorReturn"`, false, er.Index, "", ""},
	}
	for i, test := range tests {
		streaming := new(Streaming)
		handlers, paths, err := streaming.init(test.path, instance, test.name, test.tag)
		equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			continue
		}
		equal(t, streaming.pathFormatter, test.path, fmt.Sprintf("test %d", i))
		equal(t, len(handlers), 1, fmt.Sprintf("test %d", i))
		equal(t, len(paths), 1, fmt.Sprintf("test %d", i))
		equal(t, paths[0], test.path, fmt.Sprintf("test %d", i))
		sn, ok := handlers[0].(*streamingNode)
		if !ok {
			t.Errorf("not *streamingNode")
			continue
		}
		equal(t, sn.f, instance.Method(test.funcIndex), fmt.Sprintf("test %d", i))
		equal(t, fmt.Sprintf("%v", sn.requestType), test.request, fmt.Sprintf("test %d", i))
		equal(t, sn.end, test.end, fmt.Sprintf("test %d", i))
	}
}
