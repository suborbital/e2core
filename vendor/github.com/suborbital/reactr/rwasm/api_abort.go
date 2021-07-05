package rwasm

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func abortHandler() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		msgPtr := args[0].I32()
		msgSize := args[1].I32()
		filePtr := args[2].I32()
		fileSize := args[3].I32()
		lineNum := args[4].I32()
		columnNum := args[5].I32()
		ident := args[6].I32()

		return_abort(msgPtr, msgSize, filePtr, fileSize, lineNum, columnNum, ident)

		return nil, nil
	}

	return newHostFn("return_abort", 7, false, fn)
}

func return_abort(msgPtr int32, msgSize int32, filePtr int32, fileSize int32, lineNum int32, columnNum int32, ident int32) int32 {
	inst, err := instanceForIdentifier(ident, false)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	msg := inst.readMemory(msgPtr, msgSize)
	fileName := inst.readMemory(filePtr, fileSize)

	errMsg := fmt.Sprintf("runnable abort: %s; file: %s, line: %d, col: %d", msg, fileName, lineNum, columnNum)
	internalLogger.ErrorString(errMsg)

	inst.errChan <- rt.RunErr{Code: -1, Message: errMsg}

	return 0
}
