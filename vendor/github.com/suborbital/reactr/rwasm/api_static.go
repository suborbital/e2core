package rwasm

import (
	"os"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func getStaticFile() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		namePointer := args[0].I32()
		nameeSize := args[1].I32()
		ident := args[2].I32()

		ret := get_static_file(namePointer, nameeSize, ident)

		return ret, nil
	}

	return newHostFn("get_static_file", 3, true, fn)
}

func get_static_file(namePtr int32, nameSize int32, ident int32) int32 {
	inst, err := instanceForIdentifier(ident, true)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	name := inst.readMemory(namePtr, nameSize)

	file, err := inst.ctx.FileSource.GetStatic(string(name))
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] failed to GetStatic"))

		if err == rcap.ErrFileFuncNotSet {
			return -2
		} else if err == os.ErrNotExist {
			return -3
		}

		return -4
	}

	inst.setFFIResult(file)

	return int32(len(file))
}
