package rwasm

import (
	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func cacheSet() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		keyPointer := args[0].I32()
		keySize := args[1].I32()
		valPointer := args[2].I32()
		valSize := args[3].I32()
		ttl := args[4].I32()
		ident := args[5].I32()

		ret := cache_set(keyPointer, keySize, valPointer, valSize, ttl, ident)

		return ret, nil
	}

	return newHostFn("cache_set", 6, true, fn)
}

func cache_set(keyPointer int32, keySize int32, valPointer int32, valSize int32, ttl int32, identifier int32) int32 {
	inst, err := instanceForIdentifier(identifier, false)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	key := inst.readMemory(keyPointer, keySize)
	val := inst.readMemory(valPointer, valSize)

	internalLogger.Debug("[rwasm] setting cache key", string(key))

	if err := inst.ctx.Cache.Set(string(key), val, int(ttl)); err != nil {
		internalLogger.ErrorString("[rwasm] failed to set cache key", string(key), err.Error())
		return -2
	}

	return 0
}

func cacheGet() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		keyPointer := args[0].I32()
		keySize := args[1].I32()
		ident := args[2].I32()

		ret := cache_get(keyPointer, keySize, ident)

		return ret, nil
	}

	return newHostFn("cache_get", 3, true, fn)
}

func cache_get(keyPointer int32, keySize int32, identifier int32) int32 {
	inst, err := instanceForIdentifier(identifier, true)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	key := inst.readMemory(keyPointer, keySize)

	internalLogger.Debug("[rwasm] getting cache key", string(key))

	val, err := inst.ctx.Cache.Get(string(key))
	if err != nil {
		internalLogger.ErrorString("[rwasm] failed to get cache key", string(key), err.Error())
		return -2
	}

	inst.setFFIResult(val)

	return int32(len(val))
}
