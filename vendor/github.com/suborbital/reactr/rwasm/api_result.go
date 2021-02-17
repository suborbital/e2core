package rwasm

import (
	"github.com/pkg/errors"
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
	envLock.RLock()
	defer envLock.RUnlock()

	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		logger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	result := inst.readMemory(pointer, size)

	inst.resultChan <- result
}
