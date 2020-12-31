package grav

import (
	"sync"
)

// messageFilter is a series of maps that associate things about a message (its UUID, type, etc) with a boolean value to say if
// it should be allowed or denied. For each of the maps, if an entry is included, the value of the boolean is respected (true = allow, false = deny)
// Maps are either inclusive (meaning that a missing entry defaults to allow), or exclusive (meaning that a missing entry defaults to deny)
// This can be configured per map by modifiying the UUIDInclusive, TypeInclusive (etc) fields.
type messageFilter struct {
	UUIDMap       map[string]bool
	UUIDInclusive bool

	TypeMap       map[string]bool
	TypeInclusive bool

	lock sync.RWMutex
}

func newMessageFilter() *messageFilter {
	mf := &messageFilter{
		UUIDMap:       map[string]bool{},
		UUIDInclusive: true,
		TypeMap:       map[string]bool{},
		TypeInclusive: true,
		lock:          sync.RWMutex{},
	}

	return mf
}

func (mf *messageFilter) allow(msg Message) bool {
	mf.lock.RLock()
	defer mf.lock.RUnlock()

	// for each map, deny the message if:
	//	- a filter entry exists and it's value is false
	//	- a filter entry doesn't exist and its inclusive rule is false

	allowType, typeExists := mf.TypeMap[msg.Type()]
	if typeExists && !allowType {
		return false
	} else if !typeExists && !mf.TypeInclusive {
		return false
	}

	allowUUID, uuidExists := mf.UUIDMap[msg.UUID()]
	if uuidExists && !allowUUID {
		return false
	} else if !uuidExists && !mf.UUIDInclusive {
		return false
	}

	return true
}

// FilterUUID likely should not be used in normal cases, it adds a message UUID to the pod's filter.
func (mf *messageFilter) FilterUUID(uuid string, allow bool) {
	mf.lock.Lock()
	defer mf.lock.Unlock()

	mf.UUIDMap[uuid] = allow
}

func (mf *messageFilter) FilterType(msgType string, allow bool) {
	mf.lock.Lock()
	defer mf.lock.Unlock()

	mf.TypeMap[msgType] = allow
}
