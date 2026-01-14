# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy ALL source files first (including go.mod)
COPY whatsapp-mock/ ./

# Generate go.sum and download dependencies
RUN go mod tidy && go mod verify

# Build binary (go.sum now exists)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o whatsapp-mock .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/whatsapp-mock .

# Expose port
EXPOSE 8081

# Run the binary
CMD ["./whatsapp-mock"]
