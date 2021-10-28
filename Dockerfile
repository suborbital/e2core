FROM golang:1.17 as builder

RUN mkdir -p /go/src/github.com/suborbital/atmo
WORKDIR /go/src/github.com/suborbital/atmo/

# Deps first
COPY go.mod go.sum ./
RUN go mod download

# Then the rest
COPY . ./
RUN go mod vendor

# lib dance to get things building properly on ARM
RUN mkdir -p /tmp/wasmerio
RUN cp -R ./vendor/github.com/wasmerio/wasmer-go/wasmer/packaged/lib/* /tmp/wasmerio/
RUN ./scripts/copy-libs.sh
ENV LD_LIBRARY_PATH=/usr/local/lib

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
COPY --from=builder /go/src/github.com/suborbital/atmo/vendor/github.com/wasmerio/wasmer-go/wasmer/packaged/lib/ /tmp/wasmerio/

RUN /tmp/wasmerio/copy-libs.sh
ENV LD_LIBRARY_PATH=/usr/local/lib

WORKDIR /home/atmo

USER atmo