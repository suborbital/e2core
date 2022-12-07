package api

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

func (d *defaultAPI) GraphQLQueryHandler() runtime.HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		endpointPointer := args[0].(int32)
		endpointSize := args[1].(int32)
		queryPointer := args[2].(int32)
		querySize := args[3].(int32)
		ident := args[4].(int32)

		ret := d.graphqlQuery(endpointPointer, endpointSize, queryPointer, querySize, ident)

		return ret, nil
	}

	return runtime.NewHostFn("graphql_query", 5, true, fn)
}

func (d *defaultAPI) graphqlQuery(endpointPointer int32, endpointSize int32, queryPointer int32, querySize int32, identifier int32) int32 {
	inst, err := runtime.InstanceForIdentifier(identifier, true)
	if err != nil {
		runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to InstanceForIdentifier"))
		return -1
	}

	endpointBytes := inst.ReadMemory(endpointPointer, endpointSize)
	endpoint := string(endpointBytes)

	queryBytes := inst.ReadMemory(queryPointer, querySize)
	query := string(queryBytes)

	// wrap everything in a function so any errors get collected
	resp, err := func() ([]byte, error) {
		resp, err := d.capabilities.GraphQLClient.Do(d.capabilities.Auth, endpoint, query)
		if err != nil {
			runtime.InternalLogger().Error(errors.Wrap(err, "failed to GraphQLClient.Do"))
			return nil, err
		}

		respBytes, err := json.Marshal(resp)
		if err != nil {
			runtime.InternalLogger().Error(errors.Wrap(err, "[engine] alert: failed to Marshal"))
			return nil, err
		}

		return respBytes, nil
	}()

	result, err := inst.Ctx().SetFFIResult(resp, err)
	if err != nil {
		runtime.InternalLogger().ErrorString("[engine] failed to SetFFIResult", err.Error())
		return -1
	}

	return result.FFISize()
}
