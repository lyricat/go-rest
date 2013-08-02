package rest

import (
	"github.com/googollee/go-broadcast"
	"sync"
)

var rooms = make(map[string]*broadcast.Broadcast)
var roomLocker sync.RWMutex

func roomSend(id string, data interface{}) {
	roomLocker.RLock()
	room, ok := rooms[id]
	roomLocker.RUnlock()

	if !ok {
		return
	}
	room.Send(data)
}

func roomListen(id string, max int, c chan interface{}) error {
	roomLocker.Lock()
	defer roomLocker.Unlock()

	room, ok := rooms[id]
	if !ok {
		room = broadcast.NewBroadcast(max)
		rooms[id] = room
	}

	err := room.Register(c)
	if err != nil {
		return err
	}

	return nil
}

func roomUnlisten(id string, c chan interface{}) {
	roomLocker.RLock()
	defer roomLocker.RUnlock()

	room, ok := rooms[id]
	if !ok {
		return
	}

	room.Unregister(c)
	if room.Len() == 0 {
		delete(rooms, id)
	}
}
