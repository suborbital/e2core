package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

func (d *defaultAPI) GetStaticFileHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		namePointer := args[0].(int32)
		nameeSize := args[1].(int32)
		ident := args[2].(int32)

		ret := d.getStaticFile(namePointer, nameeSize, ident)

		return ret, nil
	}

	return runtime.NewHostFn("get_static_file", 3, true, fn)
}

func (d *defaultAPI) getStaticFile(namePtr int32, nameSize int32, ident int32) int32 {
	inst, err := runtime.InstanceForIdentifier(ident, true)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	name := inst.ReadMemory(namePtr, nameSize)

	file, err := d.capabilities.FileSource.GetStatic(string(name))
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] failed to GetStatic"))
	}

	result, err := inst.Ctx().SetFFIResult(file, err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[engine] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
