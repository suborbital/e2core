FROM golang:1.20 as builder
WORKDIR /go/src/github.com/suborbital/e2core/
ARG VERSION="dev"

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o .bin/e2core -tags netgo -ldflags="-extldflags=-static -X 'github.com/suborbital/e2core/e2core/release.Version=$VERSION'" .


FROM debian:buster-slim
RUN groupadd -g 999 e2core && \
	useradd -r -u 999 -g e2core e2core && \
	mkdir -p /home/e2core && \
	chown -R e2core /home/e2core && \
	chmod -R 700 /home/e2core
RUN apt-get update \
	&& apt-get install -y ca-certificates \
	&& apt-get install -y curl

# e2core binary
COPY --from=builder /go/src/github.com/suborbital/e2core/.bin/e2core /usr/local/bin/

WORKDIR /home/e2core

USER e2core
CMD ["/usr/local/bin/e2core", "start"]
