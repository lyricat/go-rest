package rest

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"reflect"
	"testing"
)

type FakeProcessor struct {
	last map[string]string
}

func (f FakeProcessor) NoInputNoOutput() {
	f.last["method"] = "NoInputNoOutput"
	f.last["input"] = ""
	f.last["output"] = ""
}

func (f FakeProcessor) NoInput() string {
	f.last["method"] = "NoInput"
	f.last["input"] = ""
	f.last["output"] = "output"
	return "output"
}

func (f FakeProcessor) NoOutput(post string) {
	f.last["method"] = "NoOutput"
	f.last["input"] = post
	f.last["output"] = ""
}

func (f FakeProcessor) Normal(post string) string {
	f.last["method"] = "Normal"
	f.last["input"] = post
	f.last["output"] = "output"
	return "output"
}

func (f FakeProcessor) HandleNode() {
	f.last["method"] = "HandleNode"
	f.last["input"] = ""
	f.last["output"] = ""
}

func (f FakeProcessor) ErrorInput(a, b int) {}

func (f FakeProcessor) ErrorOutput() (string, string) {
	return "", ""
}

func TestProcessorInit(t *testing.T) {
	type Test struct {
		path pathFormatter
		name string
		tag  reflect.StructTag

		ok        bool
		funcIndex int
		request   string
		response  string
	}
	s := new(FakeProcessor)
	instance := reflect.ValueOf(s).Elem()
	instanceType := instance.Type()
	nino, ok := instanceType.MethodByName("NoInputNoOutput")
	if !ok {
		t.Fatal("no NoInputNoOutput")
	}
	ni, ok := instanceType.MethodByName("NoInput")
	if !ok {
		t.Fatal("no NoInput")
	}
	no, ok := instanceType.MethodByName("NoOutput")
	if !ok {
		t.Fatal("no NoOutput")
	}
	n, ok := instanceType.MethodByName("Normal")
	if !ok {
		t.Fatal("no Normal")
	}
	hn, ok := instanceType.MethodByName("HandleNode")
	if !ok {
		t.Fatal("no HandleNode")
	}
	ei, ok := instanceType.MethodByName("ErrorInput")
	if !ok {
		t.Fatal("no ErrorInput")
	}
	eo, ok := instanceType.MethodByName("ErrorOutput")
	if !ok {
		t.Fatal("no ErrorOutput")
	}
	var tests = []Test{
		{"/", "", `func:"NoInputNoOutput"`, true, nino.Index, "<nil>", "<nil>"},
		{"/", "", `func:"NoInput"`, true, ni.Index, "<nil>", "string"},
		{"/", "", `func:"NoOutput"`, true, no.Index, "string", "<nil>"},
		{"/", "", `func:"Normal"`, true, n.Index, "string", "string"},
		{"/", "Node", ``, true, hn.Index, "<nil>", "<nil>"},
		{"/", "", `func:"ErrorInput"`, false, ei.Index, "", ""},
		{"/", "", `func:"ErrorOutput"`, false, eo.Index, "", ""},
	}
	for i, test := range tests {
		node := new(Processor)
		handlers, paths, err := node.init(test.path, instance, test.name, test.tag)
		assert.Equal(t, err == nil, test.ok, fmt.Sprintf("test %d error: %s", i, err))
		if !test.ok || err != nil {
			continue
		}
		assert.Equal(t, node.pathFormatter, test.path, fmt.Sprintf("test %d", i))
		assert.Equal(t, len(handlers), 1, fmt.Sprintf("test %d", i))
		assert.Equal(t, len(paths), 1, fmt.Sprintf("test %d", i))
		assert.Equal(t, paths[0], test.path, fmt.Sprintf("test %d", i))
		pn, ok := handlers[0].(*processorNode)
		if !ok {
			t.Errorf("not *processorNode")
			continue
		}
		assert.Equal(t, pn.f, instance.Method(test.funcIndex), fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", pn.requestType), test.request, fmt.Sprintf("test %d", i))
		assert.Equal(t, fmt.Sprintf("%v", pn.responseType), test.response, fmt.Sprintf("test %d", i))
	}
}
