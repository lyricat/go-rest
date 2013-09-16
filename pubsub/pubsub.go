// Package pubsub implement the Publish/Subscribe messaging paradigm
// where (citing Wikipedia) senders (publishers) are not programmed to
// send their messages to specific receivers (subscribers).
package pubsub

import (
	"errors"
	"path/filepath"
	"sync"
)

// Error of meeting max subscribe number.
var ErrMaxSubscribe = errors.New("subscription is maximum")

// Pubsub implement the Publish/Subscribe messaging paradigm.
type Pubsub struct {
	locker   sync.RWMutex
	max      int
	channels map[string][]chan interface{}
	patterns map[string][]chan interface{}
}

// New return a new Pubsub. The same name or pattern can only have max subscription. No limit if max <= 0.
func New(max int) *Pubsub {
	return &Pubsub{
		max:      max,
		channels: make(map[string][]chan interface{}),
		patterns: make(map[string][]chan interface{}),
	}
}

// Subscribe the message with specified name and send to channel c.
func (p *Pubsub) Subscribe(name string, c chan interface{}) error {
	if c == nil {
		return nil
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	if p.subscribe(p.channels, name, c) {
		return nil
	}
	return ErrMaxSubscribe
}

// Unsubscribe the channel c with specified name.
func (p *Pubsub) Unsubscribe(name string, c chan interface{}) {
	if c == nil {
		return
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	chans, ok := p.channels[name]
	if !ok {
		return
	}
	i := p.findChan(chans, c)
	if i < 0 {
		return
	}
	p.unsubscribe(p.channels, name, i)
}

// PSubscribe subscribe the message with the specified pattern and send to channel c.
// Pattern supported glob-style patterns:
//
//  - h?llo matches hello, hallo and hxllo
//  - h*llo matches hllo and heeeello
//  - h[ae]llo matches hello and hallo, but not hillo
func (p *Pubsub) PSubscribe(pattern string, c chan interface{}) error {
	if c == nil {
		return nil
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	if p.subscribe(p.patterns, pattern, c) {
		return nil
	}
	return ErrMaxSubscribe
}

// PUnsubscribe unsubscribes the channel c with the specified pattern.
func (p *Pubsub) PUnsubscribe(pattern string, c chan interface{}) {
	if c == nil {
		return
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	chans, ok := p.patterns[pattern]
	if !ok {
		return
	}
	i := p.findChan(chans, c)
	if i < 0 {
		return
	}
	p.unsubscribe(p.patterns, pattern, i)
}

// Publish a message with specifid name. Publish won't be blocked by channel receiving,
// if a channel doesn't ready when publish, it will be ignored.
func (p *Pubsub) Publish(name string, message interface{}) {
	p.locker.RLock()
	defer p.locker.RUnlock()
	if chans, ok := p.channels[name]; ok {
		for _, c := range chans {
			select {
			case c <- message:
			default:
			}
		}
	}
	for pattern, chans := range p.patterns {
		if ok, err := filepath.Match(pattern, name); err == nil && ok {
			for _, c := range chans {
				select {
				case c <- message:
				default:
				}
			}
		}
	}
}

// UnsubscribeAll unsubscribe channel c from all subscription & pattern subscription.
func (p *Pubsub) UnsubscribeAll(c chan interface{}) {
	if c == nil {
		return
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	type Find struct {
		name  string
		index int
	}
	for _, collection := range []map[string][]chan interface{}{p.channels, p.patterns} {
		var finds []Find
		for name, chans := range collection {
			if i := p.findChan(chans, c); i >= 0 {
				finds = append(finds, Find{name, i})
			}
		}
		for _, find := range finds {
			p.unsubscribe(collection, find.name, find.index)
		}
	}
}

func (p *Pubsub) subscribe(collection map[string][]chan interface{}, name string, c chan interface{}) bool {
	chans, ok := collection[name]
	if !ok {
		chans = []chan interface{}{c}
	} else {
		if p.findChan(chans, c) >= 0 {
			return true
		}
		if p.max > 0 && len(chans) >= p.max {
			return false
		}
		chans = append(chans, c)
	}
	collection[name] = chans
	return true
}

func (p *Pubsub) unsubscribe(collection map[string][]chan interface{}, name string, i int) {
	chans := collection[name]
	chans = append(chans[:i], chans[i+1:]...)
	if len(chans) == 0 {
		delete(collection, name)
	} else {
		collection[name] = chans
	}
}

func (p *Pubsub) findChan(chans []chan interface{}, c chan interface{}) int {
	for i, ch := range chans {
		if ch == c {
			return i
		}
	}
	return -1
}
