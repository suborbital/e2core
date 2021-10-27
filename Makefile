
atmo:
	go build -o .bin/atmo ./main.go

atmo/proxy:
	go build -o .bin/atmo-proxy -tags proxy ./main.go

run: atmo
	ATMO_HTTP_PORT=8080 .bin/atmo $(bundle)

run/proxy: build/proxy
	ATMO_HTTP_PORT=8080 .bin/atmo-proxy $(bundle)

atmo/docker: docker/dev
	docker run -v ${PWD}/$(dir):/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 suborbital/atmo:dev atmo --wait

atmo/proxy/docker: docker/dev/proxy
	docker run -v ${PWD}/$(dir):/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 --network=bridge suborbital/atmo-proxy:dev atmo-proxy

docker/dev:
	docker build . -t suborbital/atmo:dev

docker/dev/proxy:
	docker build . -f Dockerfile-proxy -t suborbital/atmo-proxy:dev

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

loadtest:
	go run ./testingsupport/load/load-tester.go

deps:
	go get -u -d ./...
	go mod vendor

mod/replace/reactr:
	go mod edit -replace github.com/suborbital/reactr=$(HOME)/Workspaces/suborbital/reactr

mod/replace/vektor:
	go mod edit -replace github.com/suborbital/vektor=$(HOME)/Workspaces/suborbital/vektor

.PHONY: build atmo atmo/docker docker/dev docker/dev/multi docker/publish docker/builder example-project test deps