package pooldirectory

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/nuexecutor/instancepool"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/system"
)

type ReuseLibrary struct {
	pool    map[fqmnString]*instancepool.Reuse
	lock    *sync.RWMutex
	source  system.Source
	hostAPI api.HostAPI
}

func NewReuse(clientSource system.Source) *ReuseLibrary {
	return &ReuseLibrary{
		pool:    make(map[fqmnString]*instancepool.Reuse),
		lock:    new(sync.RWMutex),
		source:  clientSource,
		hostAPI: api.New(zerolog.Nop()),
	}
}

func (l *ReuseLibrary) GetInstance(ctx context.Context, target fqmn.FQMN) (*instance.Instance, error) {
	ctx, span := tracing.Tracer.Start(ctx, "reuse library getinstnace")
	defer span.End()

	key := fqmnString(fmt.Sprintf(libraryKey, target.Tenant, target.Namespace, target.Name, target.Ref))

	l.lock.RLock()
	p, ok := l.pool[key]
	l.lock.RUnlock()

	if ok {
		span.AddEvent("found pool, using that one")

		return p.GetInstance(ctx), nil
	}

	fqmnString := fmt.Sprintf(fqmnStringKey, target.Tenant, target.Namespace, target.Name, target.Ref)

	mod, err := l.source.GetModule(fqmnString)
	if err != nil {
		return nil, errors.Wrap(err, "l.source.GetModule")
	}

	pool, err := instancepool.NewReuse(mod.WasmRef.Data, l.hostAPI, zerolog.Nop())
	if err != nil {
		return nil, errors.Wrap(err, "instancepool.NewReuse")
	}

	l.lock.Lock()
	l.pool[key] = &pool
	l.lock.Unlock()

	return pool.GetInstance(ctx), nil
}

func (l *ReuseLibrary) GiveInstanceBack(ctx context.Context, target fqmn.FQMN, i *instance.Instance) error {
	ctx, span := tracing.Tracer.Start(ctx, "reuse library GiveInstanceBack")
	defer span.End()

	key := fqmnString(fmt.Sprintf(libraryKey, target.Tenant, target.Namespace, target.Name, target.Ref))

	l.lock.RLock()
	p, ok := l.pool[key]
	l.lock.RUnlock()

	if !ok {
		span.AddEvent("give instance back failed")

		return errors.New("give instance back died because apparently the pool no longer exists for the fqmn")
	}

	p.GiveInstanceBack(ctx, i)

	return nil
}
