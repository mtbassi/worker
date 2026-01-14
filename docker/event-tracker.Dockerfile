# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy workspace go.mod first
COPY go.mod go.sum* ./

# Copy shared module
COPY shared/ ./shared/

# Copy event-tracker module
COPY event-tracker/ ./event-tracker/

# Set working directory to event-tracker
WORKDIR /app/event-tracker

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o event-tracker ./cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/event-tracker/event-tracker .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./event-tracker"]
