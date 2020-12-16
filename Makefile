
build/atmo:
	go build -o .bin/atmo ./main.go

atmo: build/atmo
	.bin/atmo $(bundle)

test/run:
	go run ./main.go

test/go:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...

.PHONY: build/atmo atmo test/run test/go deps