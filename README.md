# Batsign

A lightweight API key management system for Kubernetes that provides secure authentication through Envoy's external authorization protocol.

## Overview

Batsign replaces heavier solutions like OPA with a focused, efficient API key validation service for Envoy-based API gateways.

**Features:**
- Cryptographically secure API key generation
- SHA-256 hashed storage (no plain-text keys)
- Real-time validation via Envoy ext_authz gRPC
- Kubernetes-native with CRD-based management
- Minimal footprint (scratch-based Docker image)

**Components:**
- **Client CLI** - Generates API keys and Kubernetes manifests
- **Server** - Validates API keys via Envoy's external authorization interface

## Quick Start

### Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured
- Envoy-based API gateway (e.g., kgateway, Gloo Edge)

### Installation

```bash
# 1. Deploy the CRD
kubectl apply -f deploy/apikey-crd.yaml

# 2. Deploy the server
kubectl apply -f deploy/apikey-manager-server.yaml

# 3. Configure Envoy integration
kubectl apply -f deploy/apikey-manager-traffic-policy.yaml

# 4. Verify
kubectl get pods -n kgateway-system -l app=apikey-manager-server
```

## Usage

### Build the Client

```bash
go build -o bin/batsign-client ./cmd/client
# or: mise run build-client
```

### Generate and Apply an API Key

```bash
# Generate key and apply to Kubernetes
./bin/batsign-client -e user@example.com -d "Production key" 2>apikey.txt | kubectl apply -f -
```

The API key is saved to `apikey.txt`, while the manifest is applied to Kubernetes.

**Important:** Save the API key immediately—it cannot be retrieved later.

### Test the API Key

```bash
# Use with Authorization Bearer header
curl -H "Authorization: Bearer $(cat apikey.txt)" https://api.example.com/v1/models

# Or with x-api-key header
curl -H "x-api-key: $(cat apikey.txt)" https://api.example.com/v1/models
```

### Manage API Keys

```bash
# List all keys
kubectl get apikeys -A

# Disable a key
kubectl patch apikey <name> -p '{"spec":{"enabled":false}}' --type=merge

# Delete a key
kubectl delete apikey <name>
```

## Development

### Build

```bash
go build -o bin/batsign-client ./cmd/client
go build -o bin/batsign-server ./cmd/server
```

### Test

```bash
go test ./...
```

### Docker Build

```bash
docker build -t batsign:latest .
```

## Configuration

### Server Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--grpc-port` | 9191 | Envoy ext_authz gRPC service port |
| `--http-port` | 8080 | Health and stats endpoints port |
| `--namespace` | "" | Namespace to watch (empty = all) |
| `--log-level` | info | Logging level (debug/info/warn/error) |

### Server Endpoints

- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /stats` - Statistics (JSON)
- `GRPC :9191` - Envoy ext_authz service

## Security

- **No plain-text storage** - Keys hashed with SHA-256
- **Cryptographically secure** - Uses `crypto/rand`
- **Minimal attack surface** - Scratch-based Docker image
- **Non-root execution** - Runs as UID 65534
- **Read-only filesystem**
- **No key recovery** - By design

## License

Apache 2.0