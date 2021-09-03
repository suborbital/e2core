package rcap

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

// ErrCacheKeyNotFound is returned when a non-existent cache key is requested
var ErrCacheKeyNotFound = errors.New("key not found")

// CacheConfig is configuration for the cache capability
type CacheConfig struct {
	Enabled     bool         `json:"enabled" yaml:"enabled"`
	Rules       CacheRules   `json:"rules" yaml:"rules"`
	RedisConfig *RedisConfig `json:"redis,omitempty" yaml:"redis,omitetmpty"`
}

type CacheRules struct {
	AllowSet    bool `json:"allowSet" yaml:"allowSet"`
	AllowGet    bool `json:"allowGet" yaml:"allowGet"`
	AllowDelete bool `json:"allowDelete" yaml:"allowDelete"`
}

// CacheCapability gives Runnables access to a key/value cache
type CacheCapability interface {
	Set(key string, val []byte, ttl int) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// memoryCache is a "default" cache implementation for Reactr
type memoryCache struct {
	config CacheConfig
	values map[string]*uniqueVal

	lock sync.RWMutex
}

// this is used to 1) allow pointers and 2) ensure checks for unique values are cheaper (pointer equality)
type uniqueVal struct {
	val []byte
}

func SetupCache(config CacheConfig) CacheCapability {
	var cache CacheCapability

	if config.RedisConfig == nil {
		m := &memoryCache{
			config: config,
			values: make(map[string]*uniqueVal),
			lock:   sync.RWMutex{},
		}

		cache = m
	} else {
		r := newRedisCache(config)
		cache = r
	}

	return cache
}

func (m *memoryCache) Set(key string, val []byte, ttl int) error {
	if !m.config.Enabled || !m.config.Rules.AllowSet {
		return ErrCapabilityNotEnabled
	}

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
	if !m.config.Enabled || !m.config.Rules.AllowGet {
		return nil, ErrCapabilityNotEnabled
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	uVal, exists := m.values[key]
	if !exists {
		return nil, ErrCacheKeyNotFound
	}

	return uVal.val, nil
}

func (m *memoryCache) Delete(key string) error {
	if !m.config.Enabled || !m.config.Rules.AllowDelete {
		return ErrCapabilityNotEnabled
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	_, exists := m.values[key]
	if !exists {
		return nil
	}

	delete(m.values, key)

	return nil
}

func defaultCacheRules() CacheRules {
	c := CacheRules{
		AllowSet:    true,
		AllowGet:    true,
		AllowDelete: true,
	}

	return c
}
