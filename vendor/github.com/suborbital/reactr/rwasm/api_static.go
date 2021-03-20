package rwasm

import (
	"os"

	"github.com/pkg/errors"
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
		logger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	if inst.staticFileFunc == nil {
		logger.ErrorString("[rwasm] module attempted to access static file when no file access is available")
		return -2
	}

	name := inst.readMemory(namePtr, nameSize)

	file, err := inst.staticFileFunc(string(name))
	if err != nil {
		if err == os.ErrNotExist {
			logger.ErrorString("[rwasm] module requested static file that doesn't exist:", string(name))
			return -3
		}

		logger.ErrorString("[rwasm] failed to get static file:", err)
		return -4
	}

	inst.setFFIResult(file)

	return int32(len(file))
}
