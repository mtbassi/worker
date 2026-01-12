FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

COPY shared ./shared

COPY worker ./worker

WORKDIR /app/worker

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap ./cmd

FROM alpine:latest

COPY --from=builder /app/worker/bootstrap /bootstrap

CMD ["/bootstrap"]