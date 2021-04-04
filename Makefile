
build:
	go build -o .bin/atmo ./main.go

atmo: build
	ATMO_HTTP_PORT=8080 .bin/atmo $(bundle)

atmo/docker: docker/dev
	docker run -v ${PWD}/$(dir):/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 atmo:dev atmo --wait

docker/dev:
	docker build . -t atmo:dev

docker/dev/multi:
	docker buildx build . --platform linux/amd64,linux/arm64 -t atmo:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/atmo:$(version) --push

docker/publish/latest:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/atmo:latest --push

docker/builder:
	docker buildx create --use

example-project:
	subo build ./example-project --native

test:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...
	go mod vendor

.PHONY: build atmo atmo/docker docker/dev docker/dev/multi docker/publish docker/builder example-project test deps