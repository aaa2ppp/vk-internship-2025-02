FROM golang:1.23-alpine3.21 AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:3.21

COPY --from=builder /app/main /app/main

EXPOSE 8080

ENTRYPOINT ["/app/main"]
