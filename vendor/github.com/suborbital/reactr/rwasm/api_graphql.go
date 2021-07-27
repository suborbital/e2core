package rwasm

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func graphQLQuery() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		endpointPointer := args[0].I32()
		endpointSize := args[1].I32()
		queryPointer := args[2].I32()
		querySize := args[3].I32()
		ident := args[4].I32()

		ret := graphql_query(endpointPointer, endpointSize, queryPointer, querySize, ident)

		return ret, nil
	}

	return newHostFn("graphql_query", 5, true, fn)
}

func graphql_query(endpointPointer int32, endpointSize int32, queryPointer int32, querySize int32, identifier int32) int32 {
	inst, err := instanceForIdentifier(identifier, true)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	endpointBytes := inst.readMemory(endpointPointer, endpointSize)
	endpoint := string(endpointBytes)

	queryBytes := inst.readMemory(queryPointer, querySize)
	query := string(queryBytes)

	resp, err := inst.ctx.GraphQLClient.Do(inst.ctx.Auth, endpoint, query)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "failed to GraphQLClient.Do"))
		return -1
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: failed to Marshal"))
		return -1
	}

	inst.setFFIResult(respBytes)

	return int32(len(respBytes))
}
