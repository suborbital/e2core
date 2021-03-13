package rwasm

import (
	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func respSetHeader() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		keyPointer := args[0].I32()
		keySize := args[1].I32()
		valPointer := args[2].I32()
		valSize := args[3].I32()
		ident := args[4].I32()

		response_set_header(keyPointer, keySize, valPointer, valSize, ident)

		return nil, nil
	}

	return newHostFn("resp_set_header", 5, false, fn)
}

func response_set_header(keyPointer int32, keySize int32, valPointer int32, valSize int32, ident int32) int32 {
	inst, err := instanceForIdentifier(ident)
	if err != nil {
		logger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	if inst.request == nil {
		logger.ErrorString("[rwasm] Runnable attempted to access request when none is set")
		return -2
	}

	req := inst.request
	if req.RespHeaders == nil {
		req.RespHeaders = map[string]string{}
	}

	keyBytes := inst.readMemory(keyPointer, keySize)
	key := string(keyBytes)

	valBytes := inst.readMemory(valPointer, valSize)
	val := string(valBytes)

	req.RespHeaders[key] = val

	return 0
}
