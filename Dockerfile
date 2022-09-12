FROM golang:1.18 as builder

RUN mkdir -p /go/src/github.com/suborbital/deltav
WORKDIR /go/src/github.com/suborbital/deltav/

# Deps first
COPY go.mod go.sum ./
RUN go mod download

# Then the rest
COPY . ./
RUN go mod vendor

RUN make deltav/static

FROM debian:buster-slim

RUN groupadd -g 999 deltav && \
    useradd -r -u 999 -g deltav deltav && \
	mkdir -p /home/deltav && \
	chown -R deltav /home/deltav && \
	chmod -R 700 /home/deltav

RUN apt-get update \
	&& apt-get install -y ca-certificates

# deltav binary
COPY --from=builder /go/src/github.com/suborbital/deltav/.bin/deltav /usr/local/bin/

WORKDIR /home/deltav

USER deltav
