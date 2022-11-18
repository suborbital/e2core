package auth

import (
	"time"

	"github.com/suborbital/e2core/foundation/common"
)

// NewAuthorizationCache creates a new AuthorizationCache
func NewAuthorizationCache(ttl time.Duration) *AuthorizationCache {
	return newAuthorizationCache(common.SystemTime(), ttl)
}

// NewAuthorizationCache creates a new AuthorizationCache
func newAuthorizationCache(clock common.Clock, ttl time.Duration) *AuthorizationCache {
	store := common.NewTreeStore[*expiringContext]()
	cache := common.NewLoadingCache[*expiringContext](store)

	return &AuthorizationCache{
		clock: clock,
		ttl:   ttl,
		cache: cache,
	}
}

// AuthorizationCache caches authorization successful policy decisions for up to 10 minutes.
type AuthorizationCache struct {
	clock common.Clock
	ttl   time.Duration
	cache *common.LoadingCache[*expiringContext]
}

// Get fetches a cached result if present; otherwise it executes newFunc to obtain the result.
func (cache AuthorizationCache) Get(key string, newFunc func() (*TenantInfo, error)) (*TenantInfo, error) {
	// register newFunc if not previously known
	if !cache.cache.Check(key) {
		_ = cache.cache.Put(key, cache.loadingFunc(newFunc))
	}

	// return promptly if entry is found and valid
	entry := cache.cache.Get(key)
	if entry.Value != nil {
		if entry.Value.exp.After(cache.clock.Now()) {
			return entry.Value.ctx, entry.Error
		}
		// entry found but expired, refresh and await result
		_ = cache.cache.Refresh(key)
		entry = cache.cache.Get(key)
	}

	if entry.Error != nil {
		// reset entry state so subsequent requests run
		cache.cache.Replace(key, cache.loadingFunc(newFunc))
		return nil, entry.Error
	}

	return entry.Value.ctx, entry.Error
}

// loadingFun wraps a loader func with an expiringContext loader
func (cache AuthorizationCache) loadingFunc(inner func() (*TenantInfo, error)) func() (*expiringContext, error) {
	return func() (*expiringContext, error) {
		ctx, err := inner()
		if err != nil {
			return nil, err
		}

		return &expiringContext{
			exp: cache.clock.In(cache.ttl),
			ctx: ctx,
		}, nil
	}
}

// expiringContext wraps a value with an expiry
type expiringContext struct {
	exp time.Time
	ctx *TenantInfo
}
