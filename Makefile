
build:
	go build -o .bin/atmo ./main.go

build/docker:
	docker build . -t atmo:dev

atmo: build
	.bin/atmo $(bundle)

atmo/docker: build/docker
	docker run -v ${PWD}/$(dir):/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 atmo:dev atmo

test/run:
	go run ./main.go

test/go:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...

.PHONY: build build/docker atmo atmo/docker test/run test/go deps