package api

import (
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
)

func (d *defaultAPI) ReturnResultHandler() HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		size := args[1].(int32)
		ident := args[2].(int32)

		d.returnResult(pointer, size, ident)

		return nil, nil
	}

	return NewHostFn("return_result", 3, false, fn)
}

func (d *defaultAPI) returnResult(pointer int32, size int32, identifier int32) {
	inst, err := instance.ForIdentifier(identifier, false)
	if err != nil {
		d.logger.Err(err).Msg("instance.ForIdentifier")
		return
	}

	result := inst.ReadMemory(pointer, size)

	inst.SendExecutionResult(result, nil)
}

func (d *defaultAPI) ReturnErrorHandler() HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		code := args[0].(int32)
		pointer := args[1].(int32)
		size := args[2].(int32)
		ident := args[3].(int32)

		d.returnError(code, pointer, size, ident)

		return nil, nil
	}

	return NewHostFn("return_error", 4, false, fn)
}

func (d *defaultAPI) returnError(code int32, pointer int32, size int32, identifier int32) {
	inst, err := instance.ForIdentifier(identifier, false)
	if err != nil {
		d.logger.Err(err).Msg("instance.ForIdentifier")
		return
	}

	result := inst.ReadMemory(pointer, size)

	runErr := scheduler.RunErr{Code: int(code), Message: string(result)}

	inst.SendExecutionResult(nil, runErr)
}
