FROM golang:1.17 as builder

RUN mkdir -p /go/src/github.com/suborbital/atmo
WORKDIR /go/src/github.com/suborbital/atmo/

# Deps first
COPY go.mod go.sum ./
RUN go mod download

# Then the rest
COPY . ./

RUN make atmo

FROM debian:buster-slim

RUN groupadd -g 999 atmo && \
    useradd -r -u 999 -g atmo atmo && \
	mkdir -p /home/atmo && \
	chown -R atmo /home/atmo && \
	chmod -R 700 /home/atmo

RUN apt-get update \
	&& apt-get install -y ca-certificates

# atmo binary
COPY --from=builder /go/src/github.com/suborbital/atmo/.bin/atmo /usr/local/bin/atmo

WORKDIR /home/atmo

USER atmo