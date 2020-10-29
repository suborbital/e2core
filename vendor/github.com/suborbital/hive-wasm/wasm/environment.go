package wasm

// #include <stdlib.h>
//
// extern void return_result(void *context, int32_t pointer, int32_t size, int32_t ident);
// extern void return_result_swift(void *context, int32_t pointer, int32_t size, int32_t ident, int32_t swiftself, int32_t swifterr);
//
// extern int32_t fetch(void *context, int32_t urlPointer, int32_t urlSize, int32_t destPointer, int32_t destMaxSize, int32_t ident);
//
// extern void print(void *context, int32_t pointer, int32_t size, int32_t ident);
// extern void print_swift(void *context, int32_t pointer, int32_t size, int32_t ident, int32_t swiftself, int32_t swifterr);
import "C"

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"unsafe"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// the globally shared set of Wasm environments, accessed by UUID
var environments = map[string]*wasmEnvironment{}

// a lock to ensure the environments array is concurrency safe (didn't use sync.Map to prevent type coersion)
var envLock = sync.RWMutex{}

// the instance mapper maps a random int32 to a wasm instance to prevent malicious access to other instances via the FFI
var instanceMapper = sync.Map{}

// wasmEnvironment is an wasmEnvironment in which WASM instances run
type wasmEnvironment struct {
	Name      string
	UUID      string
	filepath  string
	raw       []byte
	instances []*wasmInstance

	// the index of the last used wasm instance
	instIndex int
	lock      sync.Mutex
}

type wasmInstance struct {
	wasmerInst wasmer.Instance
	resultChan chan []byte
	lock       sync.Mutex
}

// instanceReference is a "pointer" to the global environments array and the
// wasm instances within each environment
type instanceReference struct {
	EnvUUID   string
	InstIndex int
}

// newEnvironment creates a new environment and adds it to the shared environments array
// such that WASM instances can return data to the correct place
func newEnvironment(name string, filepath string) *wasmEnvironment {
	envLock.Lock()
	defer envLock.Unlock()

	e := &wasmEnvironment{
		Name:      name,
		UUID:      uuid.New().String(),
		filepath:  filepath,
		instances: []*wasmInstance{},
		instIndex: 0,
		lock:      sync.Mutex{},
	}

	environments[e.UUID] = e

	return e
}

// useInstance provides an instance from the environment's pool to be used
func (w *wasmEnvironment) useInstance(instFunc func(*wasmInstance, int32)) error {
	w.lock.Lock()

	if w.instIndex == len(w.instances)-1 {
		w.instIndex = 0
	} else {
		w.instIndex++
	}

	instIndex := w.instIndex
	inst := w.instances[instIndex]

	w.lock.Unlock() // now that we've acquired our instance, let the next one go

	inst.lock.Lock()
	defer inst.lock.Unlock()

	// generate a random identifier as a reference to the instance in use to
	// easily allow the Wasm module to reference itself when calling back over the FFI
	ident, err := setupNewIdentifier(w.UUID, instIndex)
	if err != nil {
		return errors.Wrap(err, "failed to setupNewIdentifier")
	}

	instFunc(inst, ident)

	removeIdentifier(ident)

	return nil
}

// addInstance adds a new WASM instance to the environment's pool
func (w *wasmEnvironment) addInstance() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.raw == nil || len(w.raw) == 0 {
		bytes, err := wasmer.ReadBytes(w.filepath)
		if err != nil {
			return errors.Wrap(err, "failed to ReadBytes")
		}

		w.raw = bytes
	}

	imports, err := wasmer.NewDefaultWasiImportObjectForVersion(wasmer.Snapshot1).Imports()
	if err != nil {
		return errors.Wrap(err, "failed to create Imports")
	}

	imports.AppendFunction("return_result", return_result, C.return_result)
	imports.AppendFunction("return_result_swift", return_result_swift, C.return_result_swift)
	imports.AppendFunction("fetch", fetch, C.fetch)
	imports.AppendFunction("print", print, C.print)
	imports.AppendFunction("print_swift", print_swift, C.print_swift)

	inst, err := wasmer.NewInstanceWithImports(w.raw, imports)
	if err != nil {
		return errors.Wrap(err, "failed to NewInstance")
	}

	instance := &wasmInstance{
		wasmerInst: inst,
		resultChan: make(chan []byte, 1),
		lock:       sync.Mutex{},
	}

	w.instances = append(w.instances, instance)

	return nil
}

// setRaw sets the raw bytes of a WASM module to be used rather than a filepath
func (w *wasmEnvironment) setRaw(raw []byte) {
	w.raw = raw
}

func setupNewIdentifier(envUUID string, instIndex int) (int32, error) {
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
			EnvUUID:   envUUID,
			InstIndex: instIndex,
		}

		instanceMapper.Store(ident, ref)

		return ident, nil
	}
}

func removeIdentifier(ident int32) {
	instanceMapper.Delete(ident)
}

func instanceForIdentifier(ident int32) (*wasmInstance, error) {
	rawRef, exists := instanceMapper.Load(ident)
	if !exists {
		return nil, errors.New("instance does not exist")
	}

	ref := rawRef.(instanceReference)

	envLock.RLock()
	defer envLock.RUnlock()

	env, exists := environments[ref.EnvUUID]
	if !exists {
		return nil, errors.New("environment does not exist")
	}

	if len(env.instances) <= ref.InstIndex-1 {
		return nil, errors.New("invalid instance index")
	}

	inst := env.instances[ref.InstIndex]

	return inst, nil
}

func randomIdentifier() (int32, error) {
	// generate a random number between 0 and the largest possible int32
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if err != nil {
		return -1, errors.Wrap(err, "failed to rand.Int")
	}

	return int32(num.Int64()), nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// below is the "hivew API" which grants capabilites to WASM runnables by routing things like network requests through the host (Go) code //
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//export return_result
func return_result(context unsafe.Pointer, pointer int32, size int32, identifier int32) {
	envLock.RLock()
	defer envLock.RUnlock()

	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		fmt.Println(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	result := inst.readMemory(pointer, size)

	inst.resultChan <- result
}

//export return_result_swift
func return_result_swift(context unsafe.Pointer, pointer int32, size int32, identifier int32, swiftself int32, swifterr int32) {
	return_result(context, pointer, size, identifier)
}

//export fetch
func fetch(context unsafe.Pointer, urlPointer int32, urlSize int32, destPointer int32, destMaxSize int32, identifier int32) int32 {
	// fetch makes a network request on bahalf of the wasm runner.
	// fetch writes the http response body into memory starting at returnBodyPointer, and the return value is a pointer to that memory
	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		fmt.Println(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	urlBytes := inst.readMemory(urlPointer, urlSize)

	urlObj, err := url.Parse(string(urlBytes))
	if err != nil {
		fmt.Println("couldn't parse URL")
		return -2
	}

	req, err := http.NewRequest(http.MethodGet, urlObj.String(), nil)
	if err != nil {
		fmt.Println("failed to build request")
		return -2
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("failed to Do request")
		return -3
	}

	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("failed to Read response body")
		return -4
	}

	if len(respBytes) <= int(destMaxSize) {
		inst.writeMemoryAtLocation(destPointer, respBytes)
	}

	return int32(len(respBytes))
}

//export print
func print(context unsafe.Pointer, pointer int32, size int32, identifier int32) {
	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		fmt.Println(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	msgBytes := inst.readMemory(pointer, size)
	msg := fmt.Sprintf("[%d]: %s", identifier, string(msgBytes))

	fmt.Println(msg)
}

//export print_swift
func print_swift(context unsafe.Pointer, pointer int32, size int32, identifier int32, x int32, y int32) {
	print(context, pointer, size, identifier)
}
