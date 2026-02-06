# Flagger REST API

The Flagger REST API provides HTTP endpoints to control canary deployments programmatically, enabling integration with UIs, CI/CD pipelines, and other automation tools.

## Table of Contents

- [Getting Started](#getting-started)
- [Authentication](#authentication)
- [API Endpoints](#api-endpoints)
- [Examples](#examples)
- [Error Handling](#error-handling)

## Getting Started

### Enabling the API Server

The REST API server is disabled by default. Enable it using the `--enable-rest-api` flag:

```bash
flagger --enable-rest-api=true --rest-api-port=8081
```

### Configuration Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--enable-rest-api` | `false` | Enable REST API server |
| `--rest-api-port` | `8081` | REST API server port |
| `--rest-api-auth` | `false` | Enable authentication for REST API |
| `--rest-api-cors` | `*` | CORS allowed origins (comma-separated) |

### Example Configuration

```bash
# Enable API with authentication
flagger \
  --enable-rest-api=true \
  --rest-api-port=8081 \
  --rest-api-auth=true \
  --rest-api-cors="https://ui.example.com,https://dashboard.example.com"
```

## Authentication

When authentication is enabled (`--rest-api-auth=true`), all API requests (except health checks) require a Bearer token in the Authorization header.

### Using Service Account Tokens

```bash
# Get service account token
TOKEN=$(kubectl get secret <service-account-secret> -o jsonpath='{.data.token}' | base64 -d)

# Make authenticated request
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/v1/canaries
```

### RBAC Permissions

The API integrates with Kubernetes RBAC. Ensure your service account has appropriate permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: flagger-api-reader
rules:
- apiGroups: ["flagger.app"]
  resources: ["canaries"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: flagger-api-writer
rules:
- apiGroups: ["flagger.app"]
  resources: ["canaries"]
  verbs: ["get", "list", "update"]
```

## API Endpoints

### Health Checks

#### GET /healthz

Returns the health status of the API server.

**Response:**
```
200 OK
```

#### GET /readyz

Returns the readiness status of the API server.

**Response:**
```
200 OK
```

### Canary Management

#### GET /api/v1/canaries

List all canaries across all namespaces.

**Response:**
```json
{
  "items": [
    {
      "metadata": {
        "name": "my-app",
        "namespace": "production",
        "createdAt": "2026-02-01T10:00:00Z"
      },
      "spec": { ... },
      "status": {
        "phase": "Progressing",
        "canaryWeight": 20,
        "failedChecks": 0,
        "iterations": 2,
        "lastTransitionTime": "2026-02-06T14:30:00Z"
      }
    }
  ],
  "total": 1
}
```

#### GET /api/v1/namespaces/{namespace}/canaries

List canaries in a specific namespace.

**Parameters:**
- `namespace` (path): Kubernetes namespace

**Response:** Same as GET /api/v1/canaries

#### GET /api/v1/namespaces/{namespace}/canaries/{name}

Get details of a specific canary.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Response:**
```json
{
  "metadata": {
    "name": "my-app",
    "namespace": "production",
    "createdAt": "2026-02-01T10:00:00Z"
  },
  "spec": {
    "targetRef": {
      "kind": "Deployment",
      "name": "my-app"
    },
    "progressDeadlineSeconds": 600,
    "analysis": {
      "interval": "1m",
      "threshold": 5,
      "maxWeight": 50,
      "stepWeight": 10
    }
  },
  "status": {
    "phase": "Progressing",
    "canaryWeight": 20,
    "failedChecks": 0,
    "iterations": 2,
    "lastTransitionTime": "2026-02-06T14:30:00Z"
  }
}
```

### Canary Operations

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/promote

Promote a canary to production.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Request Body (optional):**
```json
{
  "force": false,
  "skipAnalysis": false
}
```

**Response:**
```json
{
  "message": "Canary promotion initiated"
}
```

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/pause

Pause a canary rollout.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Response:**
```json
{
  "message": "Canary paused"
}
```

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/resume

Resume a paused canary rollout.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Response:**
```json
{
  "message": "Canary resumed"
}
```

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/rollback

Rollback a canary deployment.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Response:**
```json
{
  "message": "Canary rollback initiated"
}
```

### Metrics and Events

#### GET /api/v1/namespaces/{namespace}/canaries/{name}/metrics

Get current metrics for a canary.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Response:**
```json
{
  "canary": "production/my-app",
  "metrics": {
    "requestSuccessRate": 99.5,
    "requestDuration": 250
  }
}
```

#### GET /api/v1/namespaces/{namespace}/canaries/{name}/events

Get recent events for a canary.

**Parameters:**
- `namespace` (path): Kubernetes namespace
- `name` (path): Canary name

**Response:**
```json
{
  "items": [
    {
      "type": "Normal",
      "reason": "PromotionStarted",
      "message": "Canary promotion started",
      "timestamp": "2026-02-06T14:30:00Z"
    }
  ],
  "total": 1
}
```

## Examples

### List All Canaries

```bash
curl http://localhost:8081/api/v1/canaries
```

### Get Canary Details

```bash
curl http://localhost:8081/api/v1/namespaces/production/canaries/my-app
```

### Promote a Canary

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"skipAnalysis": true}' \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/promote
```

### Pause a Canary

```bash
curl -X POST \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/pause
```

### Resume a Canary

```bash
curl -X POST \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/resume
```

### Rollback a Canary

```bash
curl -X POST \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/rollback
```

### With Authentication

```bash
# Get token from service account
TOKEN=$(kubectl get secret flagger-token -n flagger-system -o jsonpath='{.data.token}' | base64 -d)

# Make authenticated request
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/v1/canaries
```

## Error Handling

The API returns standard HTTP status codes and JSON error responses:

### Error Response Format

```json
{
  "error": "Not Found",
  "message": "Canary production/my-app not found",
  "code": 404
}
```

### Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

## CORS Support

The API supports Cross-Origin Resource Sharing (CORS) for web applications. Configure allowed origins using the `--rest-api-cors` flag:

```bash
# Allow specific origins
flagger --enable-rest-api=true --rest-api-cors="https://ui.example.com,https://dashboard.example.com"

# Allow all origins (not recommended for production)
flagger --enable-rest-api=true --rest-api-cors="*"
```

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at [docs/openapi.yaml](openapi.yaml).

You can use this specification with tools like:
- [Swagger UI](https://swagger.io/tools/swagger-ui/)
- [Redoc](https://github.com/Redocly/redoc)
- [Postman](https://www.postman.com/)

## Integration Examples

### JavaScript/TypeScript

```typescript
const FLAGGER_API = 'http://localhost:8081';

async function listCanaries() {
  const response = await fetch(`${FLAGGER_API}/api/v1/canaries`);
  const data = await response.json();
  return data.items;
}

async function promoteCanary(namespace: string, name: string) {
  const response = await fetch(
    `${FLAGGER_API}/api/v1/namespaces/${namespace}/canaries/${name}/promote`,
    { method: 'POST' }
  );
  return await response.json();
}
```

### Python

```python
import requests

FLAGGER_API = 'http://localhost:8081'

def list_canaries():
    response = requests.get(f'{FLAGGER_API}/api/v1/canaries')
    response.raise_for_status()
    return response.json()['items']

def promote_canary(namespace, name, skip_analysis=False):
    url = f'{FLAGGER_API}/api/v1/namespaces/{namespace}/canaries/{name}/promote'
    data = {'skipAnalysis': skip_analysis}
    response = requests.post(url, json=data)
    response.raise_for_status()
    return response.json()
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

const flaggerAPI = "http://localhost:8081"

type CanaryListResponse struct {
    Items []CanaryResponse `json:"items"`
    Total int              `json:"total"`
}

func listCanaries() (*CanaryListResponse, error) {
    resp, err := http.Get(flaggerAPI + "/api/v1/canaries")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var canaries CanaryListResponse
    if err := json.NewDecoder(resp.Body).Decode(&canaries); err != nil {
        return nil, err
    }
    return &canaries, nil
}
```

## Security Considerations

1. **Enable Authentication**: Always enable authentication in production using `--rest-api-auth=true`
2. **Use HTTPS**: Deploy behind a reverse proxy with TLS/HTTPS
3. **Restrict CORS**: Limit CORS origins to trusted domains
4. **Network Policies**: Use Kubernetes NetworkPolicies to restrict access
5. **RBAC**: Configure minimal required permissions for API service accounts
6. **Rate Limiting**: Consider implementing rate limiting at the ingress/proxy level

## Troubleshooting

### API Not Responding

Check if the API is enabled:
```bash
kubectl logs -n flagger-system deployment/flagger | grep "REST API"
```

### Authentication Failures

Verify your token and RBAC permissions:
```bash
# Test token
kubectl auth can-i get canaries --as=system:serviceaccount:default:my-sa

# Check service account token
kubectl describe secret <token-secret>
```

### CORS Issues

Verify CORS configuration and check browser console for errors. Ensure the origin is in the allowed list.
