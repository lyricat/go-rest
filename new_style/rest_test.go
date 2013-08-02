package rest

import (
	"github.com/googollee/go-assert"
	"testing"
)

type PathWithSameMethod struct {
	get1 Get `path:"/:id"`
	get2 Get `path:"/:id"`
}

func (d *PathWithSameMethod) Get1(ctx Context) (string, error) { return "get1", nil }
func (d *PathWithSameMethod) Get2(ctx Context) (string, error) { return "get2", nil }

func TestPathWithSameMethod(t *testing.T) {
	_, err := New(new(PathWithSameMethod))
	assert.Equal(t, err.Error(), "method GET confict: get2, get1")
}

type DiffNameOfPath struct {
	get1 Get `path:"/:id"`
	get2 Get `path:"/:key"`
}

func (d *DiffNameOfPath) Get1(ctx Context) (string, error) { return "get1", nil }
func (d *DiffNameOfPath) Get2(ctx Context) (string, error) { return "get2", nil }

func TestDiffNameOfPath(t *testing.T) {
	_, err := New(new(DiffNameOfPath))
	assert.Equal(t, err.Error(), "different key name of one path: /:key at get2, /:id")
}

type NoHandlerPath struct {
	get1 Get `path:"/:id"`
}

func TestNoHandlerPath(t *testing.T) {
	_, err := New(new(NoHandlerPath))
	assert.Equal(t, err.Error(), "can't find method Get1 for node get1")
}

type ErrHandlerPath struct {
	get1 Get `path:"/:id"`
}

func (d *ErrHandlerPath) Get1() (string, error) { return "get1", nil }

func TestErrHandlerPath(t *testing.T) {
	_, err := New(new(ErrHandlerPath))
	assert.Contain(t, err.Error(), "init node Get1 error: ")
}

type InvalidPath struct {
	get1 Get `path:"%ZZ#"`
}

func (d *InvalidPath) Get1(ctx Context) (string, error) { return "get1", nil }

func TestInvalidPath(t *testing.T) {
	_, err := New(new(InvalidPath))
	assert.Equal(t, err.Error(), "invalid path tag \"/%ZZ#\" of node get1")
}

type OKPath struct {
	get  Get  `path:"/abc"`
	post Post `path:"/abc"`

	i int
	I int
}

func (d *OKPath) Get(ctx Context) (string, error)         { return "get", nil }
func (d *OKPath) Post(ctx Context, i int) (string, error) { return "post", nil }

func TestOKPath(t *testing.T) {
	_, err := New(new(OKPath))
	assert.Equal(t, err, nil)
}
