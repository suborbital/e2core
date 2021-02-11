
build:
	go build -o .bin/atmo ./main.go

build/docker:
	docker build . -t atmo:dev

build/docker/multi:
	docker buildx build . --platform linux/amd64,linux/arm64 -t atmo:dev

atmo: build
	.bin/atmo $(bundle)

atmo/docker: build/docker
	docker run -v ${PWD}/$(dir):/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 atmo:dev atmo

atmo/docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64,linux/arm/v7 -t suborbital/atmo:$(version) --push

docker/builder:
	docker buildx create --use

test/go:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...

.PHONY: build build/docker atmo atmo/docker test/run test/go deps