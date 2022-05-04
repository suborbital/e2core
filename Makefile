
velocity:
	go build -o .bin/velocity ./main.go

velocity/static:
	go build -o .bin/velocity -tags netgo -ldflags="-extldflags=-static" .

velocity/docker: docker/dev
	docker run -v ${PWD}/$(dir):/home/velocity -e ATMO_HTTP_PORT=8080 -p 8080:8080 suborbital/velocity:dev velocity --wait

docker/dev:
	docker build . -t suborbital/velocity:dev

docker/dev/multi:
	docker buildx build . --platform linux/amd64,linux/arm64 -t velocity:dev

docker/publish:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/velocity:$(version) --push

docker/publish/latest:
	docker buildx build . --platform linux/amd64,linux/arm64 -t suborbital/velocity:latest --push

docker/builder:
	docker buildx create --use

example-project:
	subo build ./example-project --native

test:
	go test -v --count=1 -p=1 ./...

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

.PHONY: build velocity velocity/docker docker/dev docker/dev/multi docker/publish docker/builder example-project test lint \
	lint/fix fix-imports deps
