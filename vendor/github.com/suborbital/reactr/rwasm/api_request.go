package rwasm

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func requestGetField() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		fieldType := args[0].I32()
		keyPointer := args[1].I32()
		keySize := args[2].I32()
		ident := args[3].I32()

		ret := request_get_field(fieldType, keyPointer, keySize, ident)

		return ret, nil
	}

	return newHostFn("request_get_field", 4, true, fn)
}

func request_get_field(fieldType int32, keyPointer int32, keySize int32, identifier int32) int32 {
	inst, err := instanceForIdentifier(identifier, true)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	keyBytes := inst.readMemory(keyPointer, keySize)
	key := string(keyBytes)

	val, err := inst.ctx.RequestHandler.GetField(fieldType, key)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "failed to GetField"))

		switch err {
		case rcap.ErrReqNotSet:
			return -2
		case rcap.ErrInvalidKey:
			return -3
		case rcap.ErrInvalidFieldType:
			return -4
		default:
			return -5
		}
	}

	inst.setFFIResult(val)

	return int32(len(val))
}
