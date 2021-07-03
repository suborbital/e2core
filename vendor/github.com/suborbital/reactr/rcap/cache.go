package rcap

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

// ErrCacheKeyNotFound is returned when a non-existent cache key is requested
var ErrCacheKeyNotFound = errors.New("key not found")

// Cache represents access to a persistent cache
type Cache interface {
	Set(key string, val []byte, ttl int) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// memoryCache is a "default" cache implementation for Reactr
type memoryCache struct {
	values map[string]*uniqueVal

	lock sync.RWMutex
}

// this is used to 1) allow pointers and 2) ensure checks for unique values are cheaper (pointer equality)
type uniqueVal struct {
	val []byte
}

func DefaultCache() Cache {
	m := &memoryCache{
		values: make(map[string]*uniqueVal),
		lock:   sync.RWMutex{},
	}

	return m
}

func (m *memoryCache) Set(key string, val []byte, ttl int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	uVal := &uniqueVal{
		val: val,
	}

	m.values[key] = uVal

	if ttl > 0 {
		go func() {
			<-time.After(time.Second * time.Duration(ttl))

			m.lock.Lock()
			defer m.lock.Unlock()

			currentVal := m.values[key]
			if currentVal == uVal {
				delete(m.values, key)
			}
		}()
	}

	return nil
}

func (m *memoryCache) Get(key string) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	uVal, exists := m.values[key]
	if !exists {
		return nil, ErrCacheKeyNotFound
	}

	return uVal.val, nil
}

func (m *memoryCache) Delete(key string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, exists := m.values[key]
	if !exists {
		return nil
	}

	delete(m.values, key)

	return nil
}
