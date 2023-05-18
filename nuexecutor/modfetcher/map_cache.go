package modfethcer

import (
	"context"
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/nuexecutor/worker"
	"github.com/suborbital/systemspec/fqmn"
)

const (
	modCacheExpiry = 168 * time.Hour
	refCacheExpiry = time.Minute

	// modKeyFormat expects the fqmn.URLPath() to be in the placeholder.
	modKeyFormat = "fqmn:/%s"
	refKeyFormat = "--%s/%s/%s"
)

type MapCache struct {
	embedded worker.ModSource
	cache    *bigcache.BigCache
	refCache *bigcache.BigCache
}

var _ worker.ModSource = &MapCache{}

func NewCache(ctx context.Context, embedded worker.ModSource) (*MapCache, error) {
	c, err := bigcache.New(ctx, bigcache.DefaultConfig(modCacheExpiry))
	if err != nil {
		return nil, errors.Wrap(err, "bigcache.New for modCache")
	}

	rc, err := bigcache.New(ctx, bigcache.DefaultConfig(refCacheExpiry))
	if err != nil {
		return nil, errors.Wrap(err, "bigcache.New for refCache")
	}

	return &MapCache{
		embedded: embedded,
		cache:    c,
		refCache: rc,
	}, nil
}

func (m MapCache) Get(ctx context.Context, fqmn fqmn.FQMN) ([]byte, error) {
	key := fmt.Sprintf(modKeyFormat, fqmn.URLPath())

	entry, err := m.cache.Get(key)
	if err == nil {
		return entry, nil
	}

	if !errors.Is(err, bigcache.ErrEntryNotFound) {
		// errored, errors is not the "not found", something went horribly wrong.
		return nil, errors.Wrap(err, "c.cache.Get returned an error other than 'not found'")
	}

	// errored, it's a 'not found', so ask embedded, and store in cache.
	module, err := m.embedded.Get(ctx, fqmn)
	if err != nil {
		return nil, errors.Wrap(err, "m.embedded.Get")
	}

	err = m.cache.Set(key, module)
	if err != nil {
		return nil, errors.Wrap(err, "m.cache.Set")
	}

	return module, nil
}

func (m MapCache) LatestRef(ctx context.Context, ident, namespace, name string) (string, error) {
	ctx, span := tracing.Tracer.Start(ctx, "mapcache.LatestRef")
	defer span.End()

	key := fmt.Sprintf(refKeyFormat, ident, namespace, name)

	entry, err := m.refCache.Get(key)
	if err == nil {
		return string(entry), nil
	}

	if !errors.Is(err, bigcache.ErrEntryNotFound) {
		// errored, errors is not the "not found", something went horribly wrong.
		return "", errors.Wrap(err, "c.refCache.Get returned an error other than 'not found'")
	}

	ref, err := m.embedded.LatestRef(ctx, ident, namespace, name)
	if err != nil {
		return "", errors.Wrap(err, "m.embedded.LatestRef")
	}

	err = m.refCache.Set(key, []byte(ref))
	if err != nil {
		return "", errors.Wrap(err, "m.refCache.Set")
	}

	return ref, nil
}
