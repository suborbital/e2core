package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

func (d *defaultAPI) GetFFIResultHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		ident := args[1].(int32)

		ret := d.getFfiResult(pointer, ident)

		return ret, nil
	}

	return runtime.NewHostFn("get_ffi_result", 2, true, fn)
}

func (d *defaultAPI) getFfiResult(pointer int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] failed to instanceForIdentifier"))
		return -1
	}

	result, err := inst.Ctx().UseFFIResult()
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] failed to useFFIResult"))
		return -1
	}

	data := result.Result
	if result.Err != nil {
		data = []byte(result.Err.Error())
	}

	inst.WriteMemoryAtLocation(pointer, data)

	return 0
}
