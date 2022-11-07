package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/capabilities"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

type logScope struct {
	RequestID  string `json:"request_id,omitempty"`
	Identifier int32  `json:"ident"`
}

func (d *defaultAPI) LogMsgHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		pointer := args[0].(int32)
		size := args[1].(int32)
		level := args[2].(int32)
		ident := args[3].(int32)

		d.logMsg(pointer, size, level, ident)

		return nil, nil
	}

	return runtime.NewHostFn("log_msg", 4, false, fn)
}

func (d *defaultAPI) logMsg(pointer int32, size int32, level int32, identifier int32) {
	inst, err := runtime.InstanceForIdentifier(identifier, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return
	}

	msgBytes := inst.ReadMemory(pointer, size)

	scope := logScope{Identifier: identifier}

	req := RequestFromContext(inst.Ctx().Context)

	if req != nil {
		handler := capabilities.NewRequestHandler(*d.capabilities.RequestConfig, req)

		// if this job is handling a request, add the Request ID for extra context
		requestID, err := handler.GetField(capabilities.RequestFieldTypeMeta, "id")
		if err != nil {
			// do nothing, we won't fail the log call because of this
		} else {
			scope.RequestID = string(requestID)
		}
	}

	d.capabilities.LoggerSource.Log(level, string(msgBytes), scope)
}
