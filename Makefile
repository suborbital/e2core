
build:
	go build -o .bin/atmo ./main.go

atmo: build
	.bin/atmo $(bundle)

atmo/docker: docker/dev
	docker run -v ${PWD}/$(dir):/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 atmo:dev atmo

docker/dev:
	docker build . -t atmo:dev

docker/dev/multi:
	docker buildx build . --platform linux/amd64,linux/arm64 -t atmo:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64,linux/arm/v7 -t suborbital/atmo:$(version) --push

docker/builder:
	docker buildx create --use

test/go:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...

.PHONY: build atmo atmo/docker docker/dev docker/dev/multi docker/publish docker/builder test/go deps