package engine

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
)

func BenchmarkRunnable(b *testing.B) {
	e := New()

	doWasm, _ := e.RegisterFromFile("wasm", "./testdata/hello-echo/hello-echo.wasm")

	for n := 0; n < b.N; n++ {
		res, err := doWasm("my name is joe").Then()
		if err != nil {
			b.Error(errors.Wrap(err, "failed to Then"))
		}

		if string(res.([]byte)) != "hello my name is joe" {
			b.Error(fmt.Errorf("expected 'hello my name is joe', got %s", string(res.([]byte))))
		}
	}
}

func BenchmarkSwiftRunnable(b *testing.B) {
	e := New()

	doWasm, _ := e.RegisterFromFile("wasm", "./testdata/hello-swift/hello-swift.wasm")

	for n := 0; n < b.N; n++ {
		res, err := doWasm("my name is joe").Then()
		if err != nil {
			b.Error(errors.Wrap(err, "failed to Then"))
		}

		if string(res.([]byte)) != "hello my name is joe" {
			b.Error(fmt.Errorf("expected 'hello my name is joe', got %s", string(res.([]byte))))
		}
	}
}
