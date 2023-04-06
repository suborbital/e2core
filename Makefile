
e2core:
	go build -o .bin/e2core ./main.go

e2core/install:
	go install

e2core/static:
	go build -o .bin/e2core -tags netgo -ldflags="-extldflags=-static" .

e2core/docker: docker/dev
	docker run -v ${PWD}/$(dir):/home/e2core -e e2core_HTTP_PORT=8080 -p 8080:8080 suborbital/e2core:dev e2core start ./example-project/modules.wasm.zip

docker/dev:
	docker build . -t suborbital/e2core:dev

docker/dev/multi:
	docker buildx build . --platform linux/amd64,linux/arm64 -t e2core:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/e2core:$(version) --push

docker/publish/latest:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/e2core:latest --push

docker/builder:
	docker buildx create --use

example-project:
	subo build ./example-project --native

test:
	RUN_SERVER_TESTS=true go test -v --count=1 -p=1 ./...

test/ci:
	go test -v --count=1 -p=1 ./...

lint:
	docker compose -f docker-compose-util.yaml up linter

lintfixer:
	docker compose -f docker-compose-util.yaml up lintfixer

loadtest:
	go run ./testingsupport/load/load-tester.go

local:
	docker build -f Dockerfile . -t e2core:local

.PHONY: build e2core e2core/docker docker/dev docker/dev/multi docker/publish docker/builder example-project test lint \
	lint/fix fix-imports
