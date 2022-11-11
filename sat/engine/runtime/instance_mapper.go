package runtime

import (
	"crypto/rand"
	"math"
	"math/big"
	"sync"

	"github.com/pkg/errors"
)

// the instance mapper is a global var that maps a random int32 to a wasm instance to make bi-directional FFI calls "easy"
var instanceMapper = sync.Map{}

func InstanceForIdentifier(ident int32, needsFFIResult bool) (*WasmInstance, error) {
	rawRef, exists := instanceMapper.Load(ident)
	if !exists {
		return nil, errors.New("instance does not exist")
	}

	ref := rawRef.(instanceReference)

	if needsFFIResult && ref.Inst.Ctx().HasFFIResult() {
		return nil, errors.New("cannot use instance for host call with existing call in progress")
	}

	return ref.Inst, nil
}

func setupNewIdentifier(inst *WasmInstance) (int32, error) {
	for {
		ident, err := randomIdentifier()
		if err != nil {
			return -1, errors.Wrap(err, "failed to randomIdentifier")
		}

		// ensure we don't accidentally overwrite something else
		// (however unlikely that may be)
		if _, exists := instanceMapper.Load(ident); exists {
			continue
		}

		ref := instanceReference{
			Inst: inst,
		}

		instanceMapper.Store(ident, ref)

		return ident, nil
	}
}

func removeIdentifier(ident int32) {
	instanceMapper.Delete(ident)
}

func randomIdentifier() (int32, error) {
	// generate a random number between 0 and the largest possible int32
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if err != nil {
		return -1, errors.Wrap(err, "failed to rand.Int")
	}

	return int32(num.Int64()), nil
}
