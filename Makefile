test:
	go test -v --count=1 -p=1 ./...

deps:
	go get -u -d ./...

.PHONY: test deps