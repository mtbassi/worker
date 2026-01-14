# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy workspace go.mod first
COPY go.mod go.sum* ./

# Copy shared module
COPY shared/ ./shared/

# Copy worker module
COPY worker/ ./worker/

# Set working directory to worker
WORKDIR /app/worker

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/worker/worker .

# Run the binary
CMD ["./worker"]
