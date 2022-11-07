package wasmtest

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/scheduler"

	"github.com/suborbital/e2core/sat/engine"
)

func TestWasmCacheGetSetRustToSwift(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("rust-set", "../testdata/rust-set/rust-set.wasm")
	e.RegisterFromFile("swift-get", "../testdata/swift-get/swift-get.wasm")

	setJob := scheduler.NewJob("rust-set", "very important")
	getJob := scheduler.NewJob("swift-get", "")

	_, err := e.Do(setJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to set cache value"))
		return
	}

	r2, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to get cache value"))
		return
	}

	if string(r2.([]byte)) != "very important" {
		t.Error(fmt.Errorf("did not get expected output"))
	}
}

func TestWasmCacheGetSetSwiftToRust(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("swift-set", "../testdata/swift-set/swift-set.wasm")
	e.RegisterFromFile("rust-get", "../testdata/rust-get/rust-get.wasm")

	setJob := scheduler.NewJob("swift-set", "very important")
	getJob := scheduler.NewJob("rust-get", "")

	_, err := e.Do(setJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to set cache value"))
		return
	}

	r2, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to get cache value"))
		return
	}

	if string(r2.([]byte)) != "very important" {
		t.Error(fmt.Errorf("did not get expected output"))
	}
}

func TestWasmCacheGetSetSwiftToAS(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("swift-set", "../testdata/swift-set/swift-set.wasm")
	e.RegisterFromFile("as-get", "../testdata/as-get/as-get.wasm")

	setJob := scheduler.NewJob("swift-set", "very important")
	getJob := scheduler.NewJob("as-get", "")

	_, err := e.Do(setJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to set cache value"))
		return
	}

	r2, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to get cache value"))
		return
	}

	if string(r2.([]byte)) != "very important" {
		t.Error(fmt.Errorf("did not get expected output"))
	}
}

func TestWasmCacheGetSetASToRust(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("as-set", "../testdata/as-set/as-set.wasm")
	e.RegisterFromFile("rust-get", "../testdata/rust-get/rust-get.wasm")

	setJob := scheduler.NewJob("as-set", "very important")
	getJob := scheduler.NewJob("rust-get", "")

	_, err := e.Do(setJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to set cache value"))
		return
	}

	r2, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to get cache value"))
		return
	}

	if string(r2.([]byte)) != "very important" {
		t.Error(fmt.Errorf("did not get expected output"))
	}
}

func TestWasmCacheGetSetTinyGoToRust(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("tinygo-cache", "../testdata/tinygo-cache/tinygo-cache.wasm")
	e.RegisterFromFile("rust-get", "../testdata/rust-get/rust-get.wasm")

	setJob := scheduler.NewJob("tinygo-cache", "very important")
	getJob := scheduler.NewJob("rust-get", "")

	_, err := e.Do(setJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to set cache value"))
		return
	}

	r2, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to get cache value"))
		return
	}

	if string(r2.([]byte)) != "very important" {
		t.Error(fmt.Errorf("did not get expected output"))
	}
}
