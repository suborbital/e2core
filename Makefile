
build:
	go build -o .bin/atmo ./main.go

build/docker:
	docker build . -t atmo:dev

atmo: build
	.bin/atmo $(bundle)

test/run:
	go run ./main.go

test/go:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...

.PHONY: build/atmo atmo test/run test/go deps