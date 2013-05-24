package rest

import (
	"net/http"
	"testing"
)

type RestUtil struct {
	Service

	Node Processor `path:"/node" method:"GET"`
	last map[string]interface{}
}

func (r RestUtil) HandleNode() {
	r.last["vars"] = r.Vars()
	r.last["request"] = r.Request()
	r.last["resp"] = r.responseWriter
}

func TestSetTest(t *testing.T) {
	type Test struct {
		vars map[string]string
		r    *http.Request
	}
	var tests = []Test{
		{nil, nil},
		{map[string]string{"a": "1"}, nil},
		{nil, new(http.Request)},
		{map[string]string{"a": "2"}, new(http.Request)},
	}
	for i, test := range tests {
		util := new(RestUtil)
		resp, err := SetTest(util, test.vars, test.r)
		if err != nil {
			t.Fatal(err)
		}
		equal(t, util.Vars(), test.vars, "test %d", i)
		equal(t, util.Request(), test.r, "test %d", i)
		equal(t, util.responseWriter, resp, "test %d", i)
	}
}
