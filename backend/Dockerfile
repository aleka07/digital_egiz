FROM golang:1.21-alpine AS builder

# Set necessary environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o server ./cmd/server

# Create a minimal production image
FROM alpine:3.18

# Add basic utilities and CA certificates
RUN apk --no-cache add ca-certificates curl tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server /app/

# Expose port
EXPOSE 8088

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8088/health || exit 1

# Run the application
CMD ["/app/server"]
