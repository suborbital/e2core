package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/capabilities"
	"github.com/suborbital/e2core/sat/engine/runtime"
)

func (d *defaultAPI) RespSetHeaderHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		valPointer := args[2].(int32)
		valSize := args[3].(int32)
		ident := args[4].(int32)

		d.responseSetHeader(keyPointer, keySize, valPointer, valSize, ident)

		return nil, nil
	}

	return runtime.NewHostFn("resp_set_header", 5, false, fn)
}

func (d *defaultAPI) responseSetHeader(keyPointer int32, keySize int32, valPointer int32, valSize int32, ident int32) int32 {
	inst, err := runtime.InstanceForIdentifier(ident, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	keyBytes := inst.ReadMemory(keyPointer, keySize)
	key := string(keyBytes)

	valBytes := inst.ReadMemory(valPointer, valSize)
	val := string(valBytes)

	req := RequestFromContext(inst.Ctx().Context)

	if req == nil {
		runtime.InternalLogger().ErrorString("request is not set")
	}

	handler := capabilities.NewRequestHandler(*d.capabilities.RequestConfig, req)

	if err := handler.SetResponseHeader(key, val); err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] failed to SetResponseHeader"))

		if err == capabilities.ErrReqNotSet {
			return -2
		} else {
			return -5
		}
	}

	return 0
}
