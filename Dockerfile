FROM golang:1.20 as builder

RUN mkdir -p /go/src/github.com/suborbital/e2core
WORKDIR /go/src/github.com/suborbital/e2core/

# Deps first
COPY go.mod go.sum ./
RUN go mod download

# Then the rest
COPY . ./
RUN go mod vendor

RUN make e2core/static

FROM debian:buster-slim

RUN groupadd -g 999 e2core && \
	useradd -r -u 999 -g e2core e2core && \
	mkdir -p /home/e2core && \
	chown -R e2core /home/e2core && \
	chmod -R 700 /home/e2core

RUN apt-get update \
	&& apt-get install -y ca-certificates

# e2core binary
COPY --from=builder /go/src/github.com/suborbital/e2core/.bin/e2core /usr/local/bin/

WORKDIR /home/e2core

USER e2core
CMD ["/usr/local/bin/e2core", "start"]
