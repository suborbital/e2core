package pooldirectory

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/nuexecutor/instancepool"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/system"
)

const (
	libraryKey    = "library://%s-%s-%s-%s"
	fqmnStringKey = "fqmn://%s/%s/%s@%s"
)

type fqmnString string

type Library struct {
	pool    map[fqmnString]*instancepool.Pool
	lock    *sync.RWMutex
	source  system.Source
	hostAPI api.HostAPI
}

func New(clientSource system.Source) *Library {
	return &Library{
		pool:    make(map[fqmnString]*instancepool.Pool),
		lock:    new(sync.RWMutex),
		source:  clientSource,
		hostAPI: api.New(zerolog.Nop()),
	}
}

func (l *Library) GetInstance(target fqmn.FQMN) (*instance.Instance, error) {
	key := fqmnString(fmt.Sprintf(libraryKey, target.Tenant, target.Namespace, target.Name, target.Ref))

	l.lock.RLock()
	p, ok := l.pool[key]
	l.lock.RUnlock()

	if ok {
		return p.GetInstance(), nil
	}

	fqmnString := fmt.Sprintf(fqmnStringKey, target.Tenant, target.Namespace, target.Name, target.Ref)

	mod, err := l.source.GetModule(fqmnString)
	if err != nil {
		return nil, errors.Wrap(err, "l.source.GetModule")
	}

	pool, err := instancepool.New(mod.WasmRef.Data, l.hostAPI, zerolog.Nop())
	if err != nil {
		return nil, errors.Wrap(err, "instancepool.New")
	}

	l.lock.Lock()
	l.pool[key] = &pool
	l.lock.Unlock()

	return pool.GetInstance(), nil
}
