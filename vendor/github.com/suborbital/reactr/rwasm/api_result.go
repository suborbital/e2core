package rwasm

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func returnResult() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		pointer := args[0].I32()
		size := args[1].I32()
		ident := args[2].I32()

		return_result(pointer, size, ident)

		return nil, nil
	}

	return newHostFn("return_result", 3, false, fn)
}

func return_result(pointer int32, size int32, identifier int32) {
	inst, err := instanceForIdentifier(identifier, false)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	result := inst.readMemory(pointer, size)

	inst.resultChan <- result
}

func returnError() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		code := args[0].I32()
		pointer := args[1].I32()
		size := args[2].I32()
		ident := args[3].I32()

		return_error(code, pointer, size, ident)

		return nil, nil
	}

	return newHostFn("return_error", 4, false, fn)
}

func return_error(code int32, pointer int32, size int32, identifier int32) {
	inst, err := instanceForIdentifier(identifier, false)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	result := inst.readMemory(pointer, size)

	runErr := rt.RunErr{Code: int(code), Message: string(result)}

	inst.errChan <- runErr
}
