package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
)

func (d *defaultAPI) GetSecretValueHandler() HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		size := args[1].(int32)
		ident := args[2].(int32)

		ret := d.getSecretValue(pointer, size, ident)

		return ret, nil
	}

	return NewHostFn("get_secret_value", 3, true, fn)
}

func (d *defaultAPI) getSecretValue(pointer int32, size int32, identifier int32) int32 {
	inst, err := instance.ForIdentifier(identifier, false)
	if err != nil {
		d.logger.Error(errors.Wrap(err, "[engine] alert: failed to ForIdentifier"))
		return -1
	}

	keyBytes := inst.ReadMemory(pointer, size)
	key := string(keyBytes)

	val := d.capabilities.Secrets.GetSecretValue(key)

	result, err := inst.Ctx().SetFFIResult([]byte(val), err)
	if err != nil {
		d.logger.ErrorString("[engine] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
