# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Batsign is a lightweight API key management system for Kubernetes that provides secure authentication through Envoy's external authorization protocol. It replaces heavier solutions like OPA by offering a focused, efficient API key validation service.

The system consists of two main components:
1. **Client CLI** - Generates API keys and outputs Kubernetes manifests
2. **Server** - Validates API keys through Envoy's external authorization interface

## Architecture

The architecture follows a client-server model with Kubernetes integration:
- The client generates cryptographically secure API keys and outputs Kubernetes manifests
- The server watches APIKey CRDs in Kubernetes and validates incoming requests via gRPC
- API keys are hashed using SHA-256 before storage - the server never sees plain-text keys
- The server integrates with Envoy-based API gateways (like kgateway) through the ext_authz protocol

## Key Directories and Files

- `/cmd/client/` - CLI tool for generating API keys
- `/cmd/server/` - Authorization server implementation
- `/internal/apikey/` - Core API key logic (generation, hashing, YAML generation)
- `/internal/server/` - Server implementation (gRPC, CRD watching, authorization)
- `/deploy/` - Kubernetes manifests for deploying the CRD and server
- `/Dockerfile` - Multi-stage Docker build for the server
- `/.mise.toml` - Development workflow configuration with tasks

## Development Commands

### Building
```bash
# Build both client and server
go build -o bin/batsign-client ./cmd/client
go build -o bin/batsign-server ./cmd/server

# Or with mise
mise run build
```

### Testing
```bash
# Run all tests
go test ./...

# Run with mise
mise run test

# Run Ginkgo BDD tests
ginkgo -r -v ./internal
```

### Running
```bash
# Run the server
./bin/batsign-server --help

# Run the client
./bin/batsign-client --help
```

### Docker
```bash
# Build Docker image
docker build -t batsign:latest .

# Build multi-architecture images
mise run docker-build-multiarch
```

## Key Implementation Details

### API Key Generation
- Uses crypto/rand for cryptographically secure randomness
- Keys are in format "sk-<base64-url-encoded-string>"
- Hashed with SHA-256 before storage
- Visual hints generated showing first 6 and last 2 characters

### Server Architecture
- gRPC server for Envoy ext_authz integration (port 9191)
- HTTP server for health checks and statistics (port 8080)
- Real-time watching of APIKey CRDs using Kubernetes dynamic client
- Thread-safe in-memory cache of API key hashes
- Graceful shutdown handling

### Security Features
- No plain-text storage of API keys
- Minimal attack surface (scratch-based Docker image)
- Non-root execution (UID 65534)
- Read-only filesystem
- No key recovery capability (by design)

## Deployment Process

1. Apply the APIKey CRD: `kubectl apply -f deploy/apikey-crd.yaml`
2. Deploy the server: `kubectl apply -f deploy/apikey-manager-server.yaml`
3. Configure Envoy integration: `kubectl apply -f deploy/apikey-manager-traffic-policy.yaml`