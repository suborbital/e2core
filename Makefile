
deltav:
	go build -o .bin/deltav ./main.go

deltav/install:
	go install

deltav/static:
	go build -o .bin/deltav -tags netgo -ldflags="-extldflags=-static" .

deltav/docker: docker/dev
	docker run -v ${PWD}/$(dir):/home/deltav -e DELTAV_HTTP_PORT=8080 -p 8080:8080 suborbital/deltav:dev deltav start ./example-project/modules.wasm.zip

docker/dev:
	docker build . -t suborbital/deltav:dev

docker/dev/multi:
	docker buildx build . --platform linux/amd64,linux/arm64 -t deltav:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/deltav:$(version) --push

docker/publish/latest:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/deltav:latest --push

docker/builder:
	docker buildx create --use

example-project:
	subo build ./example-project --native

test:
	RUN_SERVER_TESTS=true go test -v --count=1 -p=1 ./...

lint:
	golangci-lint run ./...

lint/fix:
	golangci-lint run --fix ./...

loadtest:
	go run ./testingsupport/load/load-tester.go

deps:
	go get -u -d ./...
	go mod vendor

mod/replace/reactr:
	go mod edit -replace github.com/suborbital/reactr=$(HOME)/Workspaces/suborbital/reactr

mod/replace/vektor:
	go mod edit -replace github.com/suborbital/vektor=$(HOME)/Workspaces/suborbital/vektor

.PHONY: build deltav deltav/docker docker/dev docker/dev/multi docker/publish docker/builder example-project test lint \
	lint/fix fix-imports deps
