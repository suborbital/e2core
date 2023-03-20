package auth

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"

	"github.com/suborbital/systemspec/system"
)

// GoCacheAuthorizer has a go-cache (see https://github.com/patrickmn/go-cache) implementation. It wraps the live Authz Authorizer implementation.
type GoCacheAuthorizer struct {
	cache    *cache.Cache
	embedded Authorizer
}

// Authorize implements the Authorizer interface for GoCacheAuthorizer.
func (g GoCacheAuthorizer) Authorize(token system.Credential, identifier, namespace, name string) (TenantInfo, error) {
	// construct key to check / retrieve / set value by.
	key, err := deriveKey(token, identifier, namespace, name)
	if err != nil {
		return TenantInfo{}, errors.Wrap(err, "deriveKey")
	}

	// grab the stored pointer for the key, if any.
	ptr, found := g.cache.Get(key)

	// there wasn't any.
	if !found {
		// ask the embedded Authorizer for the TenantInfo.
		ti, err := g.embedded.Authorize(token, identifier, namespace, name)
		if err != nil {
			return TenantInfo{}, errors.Wrap(err, "g.embedded.Authorize")
		}

		// store in the cache. This does not return error. Set pointer for performance per the documentation.
		g.cache.Set(key, &ti, cache.DefaultExpiration)

		// return a now stored TenantInfo.
		return ti, nil
	}

	// assert that the pointer stored in the cache is a pointer to a TenantInfo.
	tiPtr, ok := ptr.(*TenantInfo)
	if !ok {
		return TenantInfo{}, errors.New("ptr.(*TenantInfo) not ok: pointer stored was not for a TenantInfo")
	}

	// return the value of the pointer. TenantInfo is just a bunch of strings in a struct, passing values is fine.
	return *tiPtr, nil
}

// NewGoCacheAuthorizer returns a cache implementation of the Authorizer interface that wraps another Authorizer
// implementation (expected to be the Authz client). The error is not used, but leaving it here to match the signature
// of the Authz, and BigCache implementation constructors.
func NewGoCacheAuthorizer(embedded Authorizer) (*GoCacheAuthorizer, error) {
	return &GoCacheAuthorizer{
		cache:    cache.New(DefaultCacheTTL, DefaultCacheTTClean),
		embedded: embedded,
	}, nil
}
