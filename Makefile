
build/atmo:
	go build -o .bin/atmo ./main.go

atmo: build/atmo
	.bin/atmo $(bundle)

test/run:
	go run ./main.go

test/go:
	go test -v --count=1 -p=1 ./...

test/bundle:
	cp ../subo/examples/runnables.wasm.zip ./

deps:
	go get -u -d ./...

.PHONY: atmo test/run test/go test/bundle deps