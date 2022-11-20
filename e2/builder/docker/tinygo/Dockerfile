FROM suborbital/subo:dev as subo
FROM golang:1.18-bullseye as go

FROM debian:bullseye-slim
RUN apt-get update && apt-get -y install wget

WORKDIR /usr/local

# renovate: datasource=github-releases depName=tinygo-org/tinygo
ARG TINYGO_VERSION=0.26.0
ARG TARGETARCH

RUN wget -O tinygo.tar.gz \
    "https://github.com/tinygo-org/tinygo/releases/download/v${TINYGO_VERSION}/tinygo${TINYGO_VERSION}.linux-${TARGETARCH}.tar.gz" && \
    tar xf tinygo.tar.gz && \
    bash -c "rm -rf tinygo/src/device/{sam,stm32,nxp,nrf,avr,esp,rp}" && \
    bash -c "rm -rf tinygo/lib/{nrfx,mingw-w64,macos-minimal-sdk}" && \
    rm -rf tinygo/src/examples && \
    rm -rf tinygo.tar.gz

WORKDIR /root/runnable

COPY --from=go /usr/local/go /usr/local/
COPY --from=subo /go/bin/subo /usr/local/bin

ENV PATH="/usr/local/tinygo/bin:/usr/local/go/bin:$PATH"

RUN go mod download github.com/suborbital/reactr@latest && \
    rm -rf /go/pkg/mod/github.com/suborbital/reactr*/rwasm/testdata
