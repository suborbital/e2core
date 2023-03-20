package auth

import (
	"bytes"
	"context"
	"encoding/gob"
	"sync"

	"github.com/allegro/bigcache/v3"
	"github.com/pkg/errors"

	"github.com/suborbital/systemspec/system"
)

var DefaultConfig = bigcache.Config{
	Shards:             1,
	LifeWindow:         DefaultCacheTTL,
	CleanWindow:        DefaultCacheTTClean,
	MaxEntriesInWindow: 4096,
	MaxEntrySize:       512,  // 3 * uuidv4 with -, (36), + name + struct size as ballpark
	HardMaxCacheSize:   2048, // in MB, this is 2G
}

// BigCacheAuthorizer has a big cache (see https://github.com/allegro/bigcache) implementation. It wraps the live authz
// client, though technically we could create an infinitely deep onion of different caches.
type BigCacheAuthorizer struct {
	embedded Authorizer
	mtx      *sync.Mutex
	cache    *bigcache.BigCache
}

// Authorize method implements the Authorizer interface for the Big Cache implementation. If the cache doesn't have a
// record for the arguments, it will ask the embedded (expected to be the Authz) implementation.
func (c *BigCacheAuthorizer) Authorize(token system.Credential, identifier, namespace, name string) (TenantInfo, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	// construct key to check / retrieve / set value by.
	key, err := deriveKey(token, identifier, namespace, name)
	if err != nil {
		return TenantInfo{}, errors.Wrap(err, "deriveKey")
	}

	// create TenantInfo variable.
	var ti TenantInfo

	// check if the bigcache implementation has a record about it.
	stored, err := c.cache.Get(key)

	// something went wrong.
	if err != nil {
		// it does not have an entry, this is expected.
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			// ask the wrapped authorizer for the entry.
			ti, err = c.embedded.Authorize(token, identifier, namespace, name)
			if err != nil {
				return TenantInfo{}, errors.Wrap(err, "c.embedded.Authorize")
			}

			// encode the retrieved entry into []byte to prep for storage into bigcache.
			var b bytes.Buffer
			enc := gob.NewEncoder(&b)
			err = enc.Encode(ti)
			if err != nil {
				return TenantInfo{}, errors.Wrap(err, "enc.Encode TenantInfo into []byte using gob.Encoder")
			}

			// store the encoded [] in bigcache.
			err = c.cache.Set(key, b.Bytes())
			if err != nil {
				return TenantInfo{}, errors.Wrap(err, "c.cache.Set")
			}

			// return the TenantInfo. This is now cached going forward.
			return ti, nil
		}

		// The error was NOT an ErrEntryNotFound one, which is unexpected, so something else went wrong.
		return TenantInfo{}, errors.Wrap(err, "c.cache.Get")
	}

	// cache.Get did not return an error, so it had the data we were looking for. Let's decode it.
	dec := gob.NewDecoder(bytes.NewReader(stored))
	err = dec.Decode(&ti)
	if err != nil {
		return TenantInfo{}, errors.Wrap(err, "dec.Decode bytes found in cache into TenantInfo")
	}

	// return the cached and decoded TenantInfo.
	return ti, nil
}

// NewBigCacheAuthorizer returns a new configured BigCacheAuthorizer pointer. There are no configurations to pass in,
// the constructor takes a set of values that have already been decided based on the use case.
func NewBigCacheAuthorizer(embedded Authorizer, config bigcache.Config) (*BigCacheAuthorizer, error) {
	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		return nil, errors.Wrap(err, "bigcache.New")
	}

	return &BigCacheAuthorizer{
		embedded: embedded,
		cache:    cache,
		mtx:      &sync.Mutex{},
	}, nil
}
