package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

func (d *defaultAPI) GetSecretValueHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		size := args[1].(int32)
		ident := args[2].(int32)

		ret := d.getSecretValue(pointer, size, ident)

		return ret, nil
	}

	return runtime.NewHostFn("get_secret_value", 3, true, fn)
}

func (d *defaultAPI) getSecretValue(pointer int32, size int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	keyBytes := inst.ReadMemory(pointer, size)
	key := string(keyBytes)

	val := d.capabilities.Secrets.GetSecretValue(key)

	result, err := inst.Ctx().SetFFIResult([]byte(val), err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[engine] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
