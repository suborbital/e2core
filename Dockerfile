FROM golang:1.15 as builder

RUN mkdir -p /go/src/github.com/suborbital/atmo
COPY . /go/src/github.com/suborbital/atmo/
WORKDIR /go/src/github.com/suborbital/atmo/

RUN go install

FROM debian:buster-slim

RUN groupadd -g 999 atmo && \
    useradd -r -u 999 -g atmo atmo && \
	mkdir -p /home/atmo && \
	chown -R atmo /home/atmo && \
	chmod -R 700 /home/atmo

RUN apt-get update \
	&& apt-get install -y ca-certificates

# atmo binary
COPY --from=builder /go/bin/atmo /usr/local/bin
# script for choosing the correct library based on architecture
COPY --from=builder /go/src/github.com/suborbital/atmo/scripts/copy-libs.sh /tmp/wasmerio/copy-libs.sh
# the wasmer shared libraries
COPY --from=builder /go/pkg/mod/github.com/wasmerio/wasmer-go@v1.0.1/wasmer/packaged/lib/ /tmp/wasmerio/

RUN /tmp/wasmerio/copy-libs.sh
ENV LD_LIBRARY_PATH=/usr/local/lib

WORKDIR /home/atmo

USER atmo