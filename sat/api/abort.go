package api

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
	"github.com/suborbital/e2core/scheduler"
)

func (d *defaultAPI) AbortHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		msgPtr := args[0].(int32)
		msgSize := args[1].(int32)
		filePtr := args[2].(int32)
		fileSize := args[3].(int32)
		lineNum := args[4].(int32)
		columnNum := args[5].(int32)
		ident := args[6].(int32)

		d.returnAbort(msgPtr, msgSize, filePtr, fileSize, lineNum, columnNum, ident)

		return nil, nil
	}

	return runtime.NewHostFn("return_abort", 7, false, fn)
}

func (d *defaultAPI) returnAbort(msgPtr int32, msgSize int32, filePtr int32, fileSize int32, lineNum int32, columnNum int32, ident int32) int32 {
	inst, err := runtime.InstanceForIdentifier(ident, false)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	msg := inst.ReadMemory(msgPtr, msgSize)
	fileName := inst.ReadMemory(filePtr, fileSize)

	errMsg := fmt.Sprintf("runnable abort: %s; file: %s, line: %d, col: %d", msg, fileName, lineNum, columnNum)
	runtime.InternalLogger().ErrorString(errMsg)

	inst.SendExecutionResult(nil, scheduler.RunErr{Code: -1, Message: errMsg})

	return 0
}
