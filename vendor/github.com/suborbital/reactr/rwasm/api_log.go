package rwasm

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/wasmerio/wasmer-go/wasmer"
)

type logScope struct {
	RequestID  string `json:"request_id,omitempty"`
	Identifier int32  `json:"ident"`
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
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	msgBytes := inst.readMemory(pointer, size)

	scope := logScope{Identifier: identifier}

	// if this job is handling a request, add the Request ID for extra context
	if inst.ctx.RequestHandler != nil {
		requestID, err := inst.ctx.RequestHandler.GetField(rcap.RequestFieldTypeMeta, "id")
		if err != nil {
			// do nothing, we won't fail the log call because of this
		} else {
			scope.RequestID = string(requestID)
		}
	}

	inst.ctx.LoggerSource.Log(level, string(msgBytes), scope)
}
