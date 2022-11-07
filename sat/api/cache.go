package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

func (d *defaultAPI) CacheSetHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		valPointer := args[2].(int32)
		valSize := args[3].(int32)
		ttl := args[4].(int32)
		ident := args[5].(int32)

		ret := d.cacheSet(keyPointer, keySize, valPointer, valSize, ttl, ident)

		return ret, nil
	}

	return runtime.NewHostFn("cache_set", 6, true, fn)
}

func (d *defaultAPI) cacheSet(keyPointer int32, keySize int32, valPointer int32, valSize int32, ttl int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	key := inst.ReadMemory(keyPointer, keySize)
	val := inst.ReadMemory(valPointer, valSize)

	runtime.InternalLogger().Debug("[engine] setting cache key", string(key))

	if err := d.capabilities.Cache.Set(string(key), val, int(ttl)); err != nil {
		runtime.InternalLogger().ErrorString("[engine] failed to set cache key", string(key), err.Error())
		return -2
	}

	return 0
}

func (d *defaultAPI) CacheGetHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		ident := args[2].(int32)

		ret := d.cacheGet(keyPointer, keySize, ident)

		return ret, nil
	}

	return runtime.NewHostFn("cache_get", 3, true, fn)
}

func (d *defaultAPI) cacheGet(keyPointer int32, keySize int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, true)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	key := inst.ReadMemory(keyPointer, keySize)

	runtime.InternalLogger().Debug("[engine] getting cache key", string(key))

	val, err := d.capabilities.Cache.Get(string(key))
	if err != nil {
		runtime.InternalLogger().ErrorString("[engine] failed to get cache key", string(key), err.Error())
	}

	result, err := inst.Ctx().SetFFIResult(val, err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[engine] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
