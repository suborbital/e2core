package api

import (
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
)

func (d *defaultAPI) AddFFIVariableHandler() HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		namePtr := args[0].(int32)
		nameLen := args[1].(int32)
		valPtr := args[2].(int32)
		valLen := args[3].(int32)
		ident := args[4].(int32)

		ret := d.addFfiVar(namePtr, nameLen, valPtr, valLen, ident)

		return ret, nil
	}

	return NewHostFn("add_ffi_var", 5, true, fn)
}

func (d *defaultAPI) addFfiVar(namePtr, nameLen, valPtr, valLen, identifier int32) int32 {
	inst, err := instance.ForIdentifier(identifier, false)
	if err != nil {
		d.logger.Err(err).Str("method", "addFfiVar").Msg("instance.ForIdentifier")
		return -1
	}

	nameBytes := inst.ReadMemory(namePtr, nameLen)
	name := string(nameBytes)

	valueBytes := inst.ReadMemory(valPtr, valLen)
	value := string(valueBytes)

	inst.Ctx().AddVar(name, value)

	return 0
}
