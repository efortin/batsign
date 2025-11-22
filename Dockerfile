# Multi-stage build for batsign server
# Produces a minimal scratch-based image

# Stage 1: Builder
FROM golang:1.24-alpine AS builder

# Install ca-certificates for HTTPS (will be copied to final image)
RUN apk --no-cache add ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary with optimizations
# CGO_ENABLED=0 for static linking (required for scratch image)
# -ldflags='-s -w' to strip debug info and reduce size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-s -w -extldflags "-static"' \
    -a -installsuffix cgo \
    -o batsign-server \
    ./cmd/server

# Stage 2: Minimal runtime image
FROM scratch

# Copy CA certificates from builder (needed for Kubernetes API calls)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/batsign-server /batsign-server

# Use non-root user (nobody)
USER 65534:65534

# Expose gRPC and HTTP ports
EXPOSE 9191 8080

# Run the server
ENTRYPOINT ["/batsign-server"]
