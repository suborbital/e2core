FROM golang:1.15 as builder

RUN mkdir -p /go/src/github.com/suborbital/atmo
COPY . /go/src/github.com/suborbital/atmo/
WORKDIR /go/src/github.com/suborbital/atmo/

RUN go get -v -d ./...
RUN go mod vendor
RUN go install

FROM debian:buster-slim

RUN groupadd -g 999 atmo && \
    useradd -r -u 999 -g atmo atmo && \
	mkdir -p /home/atmo && \
	chown -R atmo /home/atmo && \
	chmod -R 700 /home/atmo

RUN apt-get update \
	&& apt-get install -y ca-certificates

COPY --from=builder /go/bin/atmo /usr/local/bin

COPY --from=builder /go/src/github.com/suborbital/atmo/vendor/github.com/wasmerio/wasmer-go/wasmer/libwasmer.so /usr/local/lib/
ENV LD_LIBRARY_PATH=/usr/local/lib

WORKDIR /home/atmo

USER atmo