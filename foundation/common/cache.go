package common

import (
	"sync"

	art "github.com/plar/go-adaptive-radix-tree"
)

const (
	// EntryInit marks a cache entry as initialized but not yet populated.
	EntryInit EntryState = iota
	// EntryPending indicates the cache entry is the process of being updated.
	EntryPending
	// EntryError indicates the last attempt to update a cache entry failed.
	EntryError
	// EntryCanceled indicates an update was attempted but canceled before completing.
	EntryCanceled
	// EntryReady indicates the cache entry is ready for use.
	EntryReady
)

type (
	// EntryState represents the current state of a CacheEntry.
	EntryState uint8

	// Value is an immutable snapshot of a CacheEntry's state.
	Value[V any] struct {
		State EntryState
		Value V
		Error error
	}

	// Entry represents a cached entry.
	Entry[V any] struct {
		cond        *sync.Cond
		state       EntryState
		loadingFunc func() (V, error)
		value       V
		err         error
	}

	// LoadingCache is an in-memory key-value store.
	// Value lifecycles are managed by the cache until their key is dropped.
	LoadingCache[V any] struct {
		lock    *sync.Mutex
		entries CacheStore[V]
	}
)

type CacheStore[V any] interface {
	Get(string) *Entry[V]
	Put(string, *Entry[V])
	Delete(string)
}

func NewMapStore[V any]() MapStore[V] {
	return MapStore[V]{
		store: make(map[string]*Entry[V]),
	}
}

type MapStore[V any] struct {
	store map[string]*Entry[V]
}

func (store MapStore[V]) Get(key string) *Entry[V] {
	return store.store[key]
}

func (store MapStore[V]) Put(key string, val *Entry[V]) {
	store.store[key] = val
}

func (store MapStore[V]) Delete(key string) {
	delete(store.store, key)
}

func NewTreeStore[V any]() TreeStore[V] {
	return TreeStore[V]{
		store: art.New(),
	}
}

type TreeStore[V any] struct {
	store art.Tree
}

func (store TreeStore[V]) Get(key string) *Entry[V] {
	if val, ok := store.store.Search(art.Key(key)); ok {
		return val.(*Entry[V])
	}

	return nil
}

func (store TreeStore[V]) Put(key string, val *Entry[V]) {
	store.store.Insert(art.Key(key), val)
}

func (store TreeStore[V]) Delete(key string) {
	store.store.Delete(art.Key(key))
}

// String returns EntryState as a string.
func (state EntryState) String() string {
	var name string
	switch state {
	case EntryInit:
		name = "[EntryState=INIT]"
	case EntryPending:
		name = "[EntryState=PENDING]"
	case EntryError:
		name = "[EntryState=ERROR]"
	case EntryCanceled:
		name = "[EntryState=CANCELED]"
	case EntryReady:
		name = "[EntryState=READY]"
	}

	return name
}

// NewLoadingCache returns a new instance of LoadingCache[V].
func NewLoadingCache[V any](store CacheStore[V]) *LoadingCache[V] {
	return &LoadingCache[V]{
		lock:    new(sync.Mutex),
		entries: store,
	}
}

// Get the Entry associated with Key, loading a new instance if unknown.
// If Entry exists and is EntryPending this call parks the caller until the update is complete.
func (cache *LoadingCache[V]) Get(key string) Value[V] {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	entry := cache.entries.Get(key)
	if entry == nil {
		return Value[V]{
			State: EntryError,
			Error: DoesNotExistError("[LoadingCache] Get"),
		}
	}

	switch entry.state {
	case EntryError:
		return Value[V]{
			State: entry.state,
			Value: entry.value,
			Error: entry.err,
		}
	case EntryInit:
		asyncLoad(entry)
		fallthrough
	case EntryPending:
		// park routine until update completes
		for entry.state == EntryPending {
			entry.cond.Wait()
		}
		fallthrough
	default: // Ready | Failed | Canceled
		return Value[V]{
			State: entry.state,
			Value: entry.value,
			Error: entry.err,
		}
	}
}

// Check returns true if a key is present in the cache, else false.
func (cache *LoadingCache[V]) Check(key string) bool {
	cache.lock.Lock()
	found := cache.entries.Get(key) != nil
	cache.lock.Unlock()

	return found
}

// Put creates a new entry in the LoadingCache using the supplied loadingFunc.
// All entries are loaded lazily meaning loadingFunc is not invoked until LoadingCache.Get(key) is called.
func (cache *LoadingCache[V]) Put(key string, loadingFunc func() (V, error)) error {
	// Optimistic look up to avoid lock contention.
	entry := cache.entries.Get(key)
	if entry != nil {
		return DuplicateEntryError("[LoadingCache] Put")
	}

	cache.lock.Lock()
	defer cache.lock.Unlock()

	// verify observed state is still valid
	entry = cache.entries.Get(key)
	if entry != nil {
		return DuplicateEntryError("[LoadingCache] Put")
	}

	cache.entries.Put(key, &Entry[V]{
		cond:        sync.NewCond(cache.lock),
		state:       EntryInit,
		loadingFunc: loadingFunc})

	return nil
}

// Replace overwrites the loadingFunc for an existing key.
// If there is no key present a new one will be created.
func (cache *LoadingCache[V]) Replace(key string, loadingFunc func() (V, error)) {
	cache.lock.Lock()

	entry := cache.entries.Get(key)
	if entry == nil {
		cache.entries.Put(key, &Entry[V]{
			cond:        sync.NewCond(cache.lock),
			state:       EntryInit,
			loadingFunc: loadingFunc,
		})
	} else {
		entry.loadingFunc = loadingFunc
		entry.state = EntryInit
	}

	cache.lock.Unlock()
}

// Refresh updates a cache Entry asynchronously.
// Calling LoadingCache.Get(key) immediately allows the caller to await the result.
func (cache *LoadingCache[V]) Refresh(key string) error {
	cache.lock.Lock()

	entry := cache.entries.Get(key)
	if entry == nil {
		cache.lock.Unlock()
		return DoesNotExistError("[LoadingCache] Pending")
	}

	asyncLoad(entry)
	cache.lock.Unlock()

	return nil
}

// Cancel cancels an update and notifies all watchers.
func (cache *LoadingCache[V]) Cancel(key string) {
	cache.lock.Lock()

	entry := cache.entries.Get(key)
	if entry == nil {
		cache.lock.Unlock()
		return
	}

	if entry.state <= EntryPending {
		entry.err = ErrCanceled
		entry.state = EntryCanceled
	}

	cache.lock.Unlock()
	entry.cond.Broadcast()
}

// Drop removes the entry associated with key from the cache.
func (cache *LoadingCache[V]) Drop(key string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	entry := cache.entries.Get(key)
	if entry == nil {
		return
	}

	cache.entries.Delete(key)
	// ensure asyncLoad does not repopulate entry on completion.
	if entry.state == EntryPending {
		entry.state = EntryCanceled
	}

	entry.cond.Broadcast()
}

// Cache lock must be held when calling asyncLoad.
func asyncLoad[V any](entry *Entry[V]) {
	if entry == nil {
		return
	}

	entry.state = EntryPending

	go func() {
		// execute potentially expensive operation without holding lock
		value, err := entry.loadingFunc()

		// reacquire lock
		entry.cond.L.Lock()

		if entry.state == EntryCanceled {
			// no-op, throw-away results
			entry.err = ErrCanceled
			entry.cond.L.Unlock()
			return
		}

		// do not override value on err
		if err == nil {
			entry.state = EntryReady
			entry.value = value
			entry.err = err
		} else {
			entry.state = EntryError
			entry.err = err
		}

		// notify watchers
		entry.cond.Broadcast()

		entry.cond.L.Unlock()
	}()
}
