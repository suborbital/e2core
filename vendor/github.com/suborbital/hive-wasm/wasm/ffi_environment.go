package wasm

// #include <stdlib.h>
//
// extern void return_result(void *context, int32_t pointer, int32_t size, int32_t ident);
// extern void return_result_swift(void *context, int32_t pointer, int32_t size, int32_t ident, int32_t swiftself, int32_t swifterr);
//
// extern int32_t fetch_url(void *context, int32_t method, int32_t urlPointer, int32_t urlSize, int32_t bodyPointer, int32_t bodySize, int32_t destPointer, int32_t destMaxSize, int32_t ident);
//
// extern int32_t cache_set(void *context, int32_t keyPointer, int32_t keySize, int32_t valPointer, int32_t valSize, int32_t ttl, int32_t ident);
// extern int32_t cache_set_swift(void *context, int32_t keyPointer, int32_t keySize, int32_t valPointer, int32_t valSize, int32_t ttl, int32_t ident, int32_t swiftself, int32_t swifterr);
//
// extern int32_t cache_get(void *context, int32_t keyPointer, int32_t keySize, int32_t destPointer, int32_t destMaxSize, int32_t ident);
// extern int32_t cache_get_swift(void *context, int32_t keyPointer, int32_t keySize, int32_t destPointer, int32_t destMaxSize, int32_t ident, int32_t swiftself, int32_t swifterr);
//
// extern void log_msg(void *context, int32_t pointer, int32_t size, int32_t level, int32_t ident);
// extern void log_msg_swift(void *context, int32_t pointer, int32_t size, int32_t level, int32_t ident, int32_t swiftself, int32_t swifterr);
import "C"

import (
	"crypto/rand"
	"math"
	"math/big"
	"sync"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"unsafe"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/hive-wasm/bundle"
	"github.com/suborbital/hive/hive"
	"github.com/suborbital/vektor/vlog"
	"github.com/wasmerio/wasmer-go/wasmer"
)

/*
 In order to allow "easy" communication of data across the FFI barrier (outbound Go -> WASM and inbound WASM -> Go), hivew provides
 an FFI API. Functions exported from a WASM module can be easily called by Go code via the Wasmer instance exports, but returning data
 to the host Go code is not quite as straightforward.

 In order to accomplish this, hivew internally keeps a set of "environments" in a singleton package var (`environments` below).
 Each environment is a container that includes the WASM module bytes, and a set of WASM instances (runtimes) to execute said module.
 The envionment object has an index referencing its place in the singleton array, and each instance has an index referencing its position within
 the environment's instance array.

 When a WASM function calls one of the FFI API functions, it includes the `ident`` value that was provided at the beginning
 of job execution, which allows hivew to look up the [env][instance] and send the result on the appropriate result channel. This is needed due to
 the way Go makes functions available on the FFI using CGO.
*/

// the globally shared set of Wasm environments, accessed by UUID
var environments = map[string]*wasmEnvironment{}

// a lock to ensure the environments array is concurrency safe (didn't use sync.Map to prevent type coersion)
var envLock = sync.RWMutex{}

// the instance mapper maps a random int32 to a wasm instance to prevent malicious access to other instances via the FFI
var instanceMapper = sync.Map{}

// the logger used by Wasm Runnables
var logger = vlog.Default()

// wasmEnvironment is an environmenr in which Wasm instances run
type wasmEnvironment struct {
	UUID      string
	ref       *bundle.WasmModuleRef
	instances []*wasmInstance

	// the index of the last used wasm instance
	instIndex int
	lock      sync.Mutex
}

type wasmInstance struct {
	wasmerInst wasmer.Instance
	hiveCtx    *hive.Ctx
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
// such that Wasm instances can return data to the correct place
func newEnvironment(ref *bundle.WasmModuleRef) *wasmEnvironment {
	envLock.Lock()
	defer envLock.Unlock()

	e := &wasmEnvironment{
		UUID:      uuid.New().String(),
		ref:       ref,
		instances: []*wasmInstance{},
		instIndex: 0,
		lock:      sync.Mutex{},
	}

	environments[e.UUID] = e

	return e
}

// useInstance provides an instance from the environment's pool to be used
func (w *wasmEnvironment) useInstance(ctx *hive.Ctx, instFunc func(*wasmInstance, int32)) error {
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

	inst.hiveCtx = ctx

	// generate a random identifier as a reference to the instance in use to
	// easily allow the Wasm module to reference itself when calling back over the FFI
	ident, err := setupNewIdentifier(w.UUID, instIndex)
	if err != nil {
		return errors.Wrap(err, "failed to setupNewIdentifier")
	}

	instFunc(inst, ident)

	removeIdentifier(ident)
	inst.hiveCtx = nil

	return nil
}

// addInstance adds a new Wasm instance to the environment's pool
func (w *wasmEnvironment) addInstance() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	module, err := w.ref.ModuleBytes()
	if err != nil {
		return errors.Wrap(err, "failed to ModuleBytes")
	}

	// mount the WASI interface
	imports, err := wasmer.NewDefaultWasiImportObjectForVersion(wasmer.Snapshot1).Imports()
	if err != nil {
		return errors.Wrap(err, "failed to create Imports")
	}

	// Mount the Runnable API
	imports.AppendFunction("return_result", return_result, C.return_result)
	imports.AppendFunction("return_result_swift", return_result_swift, C.return_result_swift)

	imports.AppendFunction("fetch_url", fetch_url, C.fetch_url)

	imports.AppendFunction("cache_set", cache_set, C.cache_set)
	imports.AppendFunction("cache_set_swift", cache_set_swift, C.cache_set_swift)

	imports.AppendFunction("cache_get", cache_get, C.cache_get)
	imports.AppendFunction("cache_get_swift", cache_get_swift, C.cache_get_swift)

	imports.AppendFunction("log_msg", log_msg, C.log_msg)
	imports.AppendFunction("log_msg_swift", log_msg_swift, C.log_msg_swift)

	inst, err := wasmer.NewInstanceWithImports(module, imports)
	if err != nil {
		return errors.Wrap(err, "failed to NewInstance")
	}

	// if the module has exported an init, call it
	init := inst.Exports["init"]
	if init != nil {
		if _, err := init(); err != nil {
			return errors.Wrap(err, "failed to init instance")
		}
	}

	instance := &wasmInstance{
		wasmerInst: inst,
		resultChan: make(chan []byte, 1),
		lock:       sync.Mutex{},
	}

	w.instances = append(w.instances, instance)

	return nil
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

/////////////////////////////////////////////////////////////////////////////
// below is the wasm glue code used to manipulate wasm instance memory     //
// this requires a set of functions to be available within the wasm module //
// - allocate                                                              //
// - deallocate                                                            //
/////////////////////////////////////////////////////////////////////////////

func (w *wasmInstance) readMemory(pointer int32, size int32) []byte {
	data := w.wasmerInst.Memory.Data()[pointer:]
	result := make([]byte, size)

	for index := 0; int32(index) < size; index++ {
		result[index] = data[index]
	}

	return result
}

func (w *wasmInstance) writeMemory(data []byte) (int32, error) {
	lengthOfInput := len(data)

	allocate := w.wasmerInst.Exports["allocate"]
	if allocate == nil {
		return -1, errors.New("missing required FFI function: allocate")
	}

	// Allocate memory for the input, and get a pointer to it.
	allocateResult, err := allocate(lengthOfInput)
	if err != nil {
		return -1, errors.Wrap(err, "failed to call allocate")
	}

	pointer := allocateResult.ToI32()

	w.writeMemoryAtLocation(pointer, data)

	return pointer, nil
}

func (w *wasmInstance) writeMemoryAtLocation(pointer int32, data []byte) {
	lengthOfInput := len(data)

	// Write the input into the memory.
	memory := w.wasmerInst.Memory.Data()[pointer:]

	for index := 0; index < lengthOfInput; index++ {
		memory[index] = data[index]
	}
}

func (w *wasmInstance) deallocate(pointer int32, length int) {
	dealloc := w.wasmerInst.Exports["deallocate"]

	dealloc(pointer, length)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// below is the "Runnable API" which grants capabilites to Wasm runnables by routing things like network requests through the host (Go) code //
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

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

const (
	methodGet    = int32(1)
	methodPost   = int32(2)
	methodPatch  = int32(3)
	methodDelete = int32(4)
)

var methodValToMethod = map[int32]string{
	methodGet:    http.MethodGet,
	methodPost:   http.MethodPost,
	methodPatch:  http.MethodPatch,
	methodDelete: http.MethodDelete,
}

//export fetch_url
func fetch_url(context unsafe.Pointer, method int32, urlPointer int32, urlSize int32, bodyPointer int32, bodySize int32, destPointer int32, destMaxSize int32, identifier int32) int32 {
	// fetch makes a network request on bahalf of the wasm runner.
	// fetch writes the http response body into memory starting at returnBodyPointer, and the return value is a pointer to that memory
	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		fmt.Println(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	httpMethod, exists := methodValToMethod[method]
	if !exists {
		fmt.Println("invalid method provided")
		return -2
	}

	urlBytes := inst.readMemory(urlPointer, urlSize)

	urlObj, err := url.Parse(string(urlBytes))
	if err != nil {
		fmt.Println("couldn't parse URL")
		return -2
	}

	req, err := http.NewRequest(httpMethod, urlObj.String(), nil)
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

//export cache_set
func cache_set(context unsafe.Pointer, keyPointer int32, keySize int32, valPointer int32, valSize int32, ttl int32, identifier int32) int32 {
	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		fmt.Println(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	key := inst.readMemory(keyPointer, keySize)
	val := inst.readMemory(valPointer, valSize)

	fmt.Println("setting cache key", string(key))

	if err := inst.hiveCtx.Cache.Set(string(key), val, int(ttl)); err != nil {
		fmt.Println("failed to set cache key", string(key), err.Error())
		return -2
	}

	return 0
}

//export cache_set_swift
func cache_set_swift(context unsafe.Pointer, keyPointer int32, keySize int32, valPointer int32, valSize int32, ttl int32, identifier int32, swiftself int32, swifterr int32) int32 {
	return cache_set(context, keyPointer, keySize, valPointer, valSize, ttl, identifier)
}

//export cache_get
func cache_get(context unsafe.Pointer, keyPointer int32, keySize int32, destPointer int32, destMaxSize int32, identifier int32) int32 {
	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		fmt.Println(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	key := inst.readMemory(keyPointer, keySize)

	fmt.Println("getting cache key", string(key))

	val, err := inst.hiveCtx.Cache.Get(string(key))
	if err != nil {
		fmt.Println("failed to get cache key", key, err.Error())
		return -2
	}

	valBytes := []byte(val)

	if len(valBytes) <= int(destMaxSize) {
		inst.writeMemoryAtLocation(destPointer, valBytes)
	}

	return int32(len(valBytes))
}

//export cache_get_swift
func cache_get_swift(context unsafe.Pointer, keyPointer int32, keySize int32, destPointer int32, destMaxSize int32, identifier int32, swiftself int32, swifterr int32) int32 {
	return cache_get(context, keyPointer, keySize, destPointer, destMaxSize, identifier)
}

type logScope struct {
	Identifier int32 `json:"ident"`
}

//export log_msg
func log_msg(context unsafe.Pointer, pointer int32, size int32, level int32, identifier int32) {
	inst, err := instanceForIdentifier(identifier)
	if err != nil {
		logger.Error(errors.Wrap(err, "[hive-wasm] alert: invalid identifier used, potential malicious activity"))
		return
	}

	msgBytes := inst.readMemory(pointer, size)

	l := logger.CreateScoped(logScope{Identifier: identifier})

	switch level {
	case 1:
		l.ErrorString(string(msgBytes))
	case 2:
		l.Warn(string(msgBytes))
	default:
		l.Info(string(msgBytes))
	}
}

//export log_msg_swift
func log_msg_swift(context unsafe.Pointer, pointer int32, size int32, level int32, identifier int32, swiftself int32, swifterr int32) {
	log_msg(context, pointer, size, level, identifier)
}
