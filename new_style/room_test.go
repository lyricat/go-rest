package rest

import (
	"github.com/googollee/go-assert"
	"testing"
	"time"
)

func TestRoom(t *testing.T) {
	c1 := make(chan interface{})
	roomUnlisten("room", c1)
	err := roomListen("room", 2, c1)
	assert.MustEqual(t, err, nil)
	err = roomListen("room", 2, c1)
	assert.MustEqual(t, err, nil)
	err = roomListen("room", 2, c1)
	assert.MustEqual(t, err, nil)

	c2 := make(chan interface{})
	err = roomListen("room", 2, c2)
	assert.MustEqual(t, err, nil)

	c3 := make(chan interface{})
	err = roomListen("room", 2, c3)
	assert.MustNotEqual(t, err, nil)

	go func() {
		a1 := <-c1
		assert.MustEqual(t, a1, "abc")
		c3 <- 1
	}()

	go func() {
		a2 := <-c2
		assert.MustEqual(t, a2, "abc")
		c3 <- 1
	}()

	time.Sleep(time.Second / 10)
	roomSend("room", "abc")
	<-c3
	<-c3

	roomSend("nonexist", "abc")
}
