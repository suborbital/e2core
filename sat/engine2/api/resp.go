package api

import (
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/systemspec/capabilities"
)

func (d *defaultAPI) RespSetHeaderHandler() HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		keyPointer := args[0].(int32)
		keySize := args[1].(int32)
		valPointer := args[2].(int32)
		valSize := args[3].(int32)
		ident := args[4].(int32)

		d.responseSetHeader(keyPointer, keySize, valPointer, valSize, ident)

		return nil, nil
	}

	return NewHostFn("resp_set_header", 5, false, fn)
}

func (d *defaultAPI) responseSetHeader(keyPointer int32, keySize int32, valPointer int32, valSize int32, ident int32) int32 {
	ll := d.logger.With().Str("method", "responseSetHeader").Logger()

	inst, err := instance.ForIdentifier(ident, false)
	if err != nil {
		ll.Err(err).Msg("instance.ForIdentifier")
		return -1
	}

	keyBytes := inst.ReadMemory(keyPointer, keySize)
	key := string(keyBytes)

	valBytes := inst.ReadMemory(valPointer, valSize)
	val := string(valBytes)

	req := RequestFromContext(inst.Ctx().Context)

	if req == nil {
		ll.Error().Msg("request is not set")
	}

	handler := capabilities.NewRequestHandler(*d.capabilities.RequestConfig, req)

	if err := handler.SetResponseHeader(key, val); err != nil {
		ll.Err(err).Msg("handler.SetResponseHeader")

		if err == capabilities.ErrReqNotSet {
			return -2
		} else {
			return -5
		}
	}

	return 0
}
