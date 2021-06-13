FROM golang

WORKDIR /peerdiscovery
COPY . .
RUN go build ./examples/ipv4/main.go

CMD ["/peerdiscovery/main"]
