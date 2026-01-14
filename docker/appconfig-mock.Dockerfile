# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy ALL source files first (including go.mod)
COPY appconfig-mock/ ./

# Generate go.sum and download dependencies
RUN go mod tidy && go mod verify

# Build binary (go.sum now exists)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o appconfig-mock .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/appconfig-mock .

# Expose port
EXPOSE 2772

# Run the binary
CMD ["./appconfig-mock"]
