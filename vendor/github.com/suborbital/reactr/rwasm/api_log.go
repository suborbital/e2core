package rwasm

import (
	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

type logScope struct {
	Identifier int32 `json:"ident"`
}

func logMsg() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		pointer := args[0].I32()
		size := args[1].I32()
		level := args[2].I32()
		ident := args[3].I32()

		log_msg(pointer, size, level, ident)

		return nil, nil
	}

	return newHostFn("log_msg", 4, false, fn)
}

func log_msg(pointer int32, size int32, level int32, identifier int32) {
	inst, err := instanceForIdentifier(identifier, false)
	if err != nil {
		logger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	msgBytes := inst.readMemory(pointer, size)

	l := logger.CreateScoped(logScope{Identifier: identifier})

	switch level {
	case 1:
		l.ErrorString(string(msgBytes))
	case 2:
		l.Warn(string(msgBytes))
	case 4:
		l.Debug(string(msgBytes))
	default:
		l.Info(string(msgBytes))
	}
}
