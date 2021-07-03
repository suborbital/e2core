package rwasm

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
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
	inst, err := instanceForIdentifier(ident, false)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	keyBytes := inst.readMemory(keyPointer, keySize)
	key := string(keyBytes)

	valBytes := inst.readMemory(valPointer, valSize)
	val := string(valBytes)

	if err := inst.ctx.RequestHandler.SetResponseHeader(key, val); err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] failed to SetResponseHeader"))

		if err == rcap.ErrReqNotSet {
			return -2
		} else {
			return -5
		}
	}

	return 0
}
