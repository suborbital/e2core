FROM suborbital/subo:dev as subo

FROM ghcr.io/swiftwasm/swift:focal
WORKDIR /root/runnable
COPY --from=subo /go/bin/subo /usr/local/bin
