FROM golang:1.18 as builder
WORKDIR /go/sat

COPY go.* ./
RUN go mod download

COPY Makefile .
COPY *.go ./
COPY ./sat ./sat
COPY ./engine ./engine
COPY ./api ./api
RUN make sat/static


FROM gcr.io/distroless/static-debian11
COPY --from=builder /go/sat/.bin/sat /usr/local/bin/
RUN echo "sat:*:0:0:sat:/tmp:*" >> /etc/passwd

ENV PATH=/usr/local/bin
USER sat
