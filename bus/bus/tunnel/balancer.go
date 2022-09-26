package tunnel

import "sync"

// Balancer is a 'load balancer' for a list of UUIDs
// each time 'Next' is called, a round-robin UUID from the list
// is returned, the order is gobal and concurrency safe
type Balancer struct {
	uuids []string
	index int
	lock  sync.Mutex
}

// NewBalancer creates a new Balancer
func NewBalancer() *Balancer {
	b := &Balancer{
		uuids: []string{},
		index: 0,
		lock:  sync.Mutex{},
	}

	return b
}

// Add adds a UUID to the list
func (b *Balancer) Add(uuid string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.uuids = append(b.uuids, uuid)
}

// Remove removes a UUID from the list if it exists
// it re-creates the UUID list from scratch with a fresh array
func (b *Balancer) Remove(uuid string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	newUUIDs := []string{}

	for i := range b.uuids {
		if b.uuids[i] == uuid {
			continue
		}

		newUUIDs = append(newUUIDs, b.uuids[i])
	}

	b.uuids = newUUIDs

	// ensure the index isn't too large
	if b.index >= len(b.uuids)-1 {
		b.index = len(b.uuids) / 2
	}
}

// Next returns the next round-robin UUID from the list
func (b *Balancer) Next() string {
	b.lock.Lock()
	defer b.lock.Unlock()

	uuid := b.uuids[b.index]

	if b.index >= len(b.uuids)-1 {
		b.index = 0
	} else {
		b.index++
	}

	return uuid
}
