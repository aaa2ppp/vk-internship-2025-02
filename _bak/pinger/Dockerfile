FROM golang:1.23-alpine as builder
WORKDIR /build
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -o ping cmd/ping/ping.go
FROM alpine:3
COPY --from=builder build/ping /app/ping
COPY cmd/ping/addrs.lst /app/addrs.lst
ENTRYPOINT ["/bin/sh", "-c", "/app/ping < /app/addrs.lst"]
