//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/db/query"
)

func DbExec(queryType query.QueryType, name string, arguments []query.Argument) ([]byte, error) {
	ptr, size := rawSlicePointer([]byte(name))

	for _, arg := range arguments {
		addVar(arg.Name, arg.Value)
	}

	return result(C.db_exec(int32(queryType), ptr, size, Ident()))
}
