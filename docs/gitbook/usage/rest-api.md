# Flagger REST API

Flagger provides a REST API for controlling and monitoring canary deployments programmatically. This allows integration with UIs, CI/CD pipelines, and other automation tools.

## Enabling the API

The API server is disabled by default. To enable it, use the following flags:

```bash
flagger \
  -enable-api=true \
  -api-port=8081 \
  -api-token=your-secret-token \
  -api-cors-origins="https://your-ui.example.com,https://another-ui.example.com"
```

### Configuration Flags

- `--enable-api`: Enable the REST API server (default: false)
- `--api-port`: Port for the API server (default: 8081)
- `--api-token`: Bearer token for authentication. If empty, authentication is disabled
- `--api-cors-origins`: Comma-separated list of allowed CORS origins. Use `*` to allow all origins

### Environment Variables

You can also configure the API using environment variables:

- `API_TOKEN`: Bearer token for authentication

## Authentication

The API uses Bearer token authentication. Include the token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer your-secret-token" \
  http://localhost:8081/api/v1/canaries
```

Health check endpoints (`/healthz` and `/readyz`) do not require authentication.

## API Endpoints

### Health Checks

#### GET /healthz

Returns the health status of the API server.

```bash
curl http://localhost:8081/healthz
```

Response: `OK`

#### GET /readyz

Returns the readiness status of the API server.

```bash
curl http://localhost:8081/readyz
```

Response: `OK`

### Canary Management

#### GET /api/v1/canaries

List all canaries across all namespaces.

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/canaries
```

Response:
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

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries
```

#### GET /api/v1/namespaces/{namespace}/canaries/{name}

Get details of a specific canary.

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app
```

Response:
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

#### GET /api/v1/namespaces/{namespace}/canaries/{name}/metrics

Get metrics for a specific canary.

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/metrics
```

#### GET /api/v1/namespaces/{namespace}/canaries/{name}/events

Get events for a specific canary.

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/events
```

Response:
```json
{
  "events": [
    {
      "type": "Normal",
      "reason": "Synced",
      "message": "Canary deployment initialized",
      "timestamp": "2026-02-06T14:30:00Z",
      "source": "flagger"
    }
  ],
  "total": 1
}
```

### Canary Control

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/promote

Promote a canary to production.

```bash
curl -X POST \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"skipAnalysis": true}' \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/promote
```

Request body (optional):
```json
{
  "force": false,
  "skipAnalysis": false
}
```

Response:
```json
{
  "success": true,
  "message": "Canary promotion initiated"
}
```

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/pause

Pause a canary rollout.

```bash
curl -X POST \
  -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/pause
```

Response:
```json
{
  "success": true,
  "message": "Canary paused"
}
```

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/resume

Resume a paused canary rollout.

```bash
curl -X POST \
  -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/resume
```

Response:
```json
{
  "success": true,
  "message": "Canary resumed"
}
```

#### POST /api/v1/namespaces/{namespace}/canaries/{name}/rollback

Rollback a canary deployment.

```bash
curl -X POST \
  -H "Authorization: Bearer your-token" \
  http://localhost:8081/api/v1/namespaces/production/canaries/my-app/rollback
```

Response:
```json
{
  "success": true,
  "message": "Canary rollback initiated"
}
```

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": "Not Found",
  "message": "Canary production/my-app not found",
  "code": 404
}
```

HTTP Status Codes:
- `200` - Success
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `500` - Internal Server Error

## OpenAPI Specification

The full OpenAPI/Swagger specification is available in [docs/openapi.yaml](openapi.yaml).

You can use tools like Swagger UI or Postman to explore the API interactively.

## Security Considerations

1. **Authentication**: Always configure a strong bearer token in production environments
2. **TLS/HTTPS**: Use a reverse proxy (e.g., nginx, Traefik) to add TLS termination
3. **CORS**: Configure `api-cors-origins` to restrict access to trusted domains only
4. **RBAC**: The API uses the same Kubernetes service account as Flagger, so RBAC policies apply
5. **Network Policies**: Consider using Kubernetes NetworkPolicies to restrict access to the API port

## Examples

### JavaScript/TypeScript

```typescript
const API_URL = 'http://localhost:8081';
const API_TOKEN = 'your-secret-token';

async function listCanaries() {
  const response = await fetch(`${API_URL}/api/v1/canaries`, {
    headers: {
      'Authorization': `Bearer ${API_TOKEN}`
    }
  });
  return response.json();
}

async function promoteCanary(namespace: string, name: string) {
  const response = await fetch(
    `${API_URL}/api/v1/namespaces/${namespace}/canaries/${name}/promote`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ skipAnalysis: false })
    }
  );
  return response.json();
}
```

### Python

```python
import requests

API_URL = 'http://localhost:8081'
API_TOKEN = 'your-secret-token'

headers = {
    'Authorization': f'Bearer {API_TOKEN}'
}

def list_canaries():
    response = requests.get(f'{API_URL}/api/v1/canaries', headers=headers)
    return response.json()

def promote_canary(namespace, name, skip_analysis=False):
    url = f'{API_URL}/api/v1/namespaces/{namespace}/canaries/{name}/promote'
    data = {'skipAnalysis': skip_analysis}
    response = requests.post(url, headers=headers, json=data)
    return response.json()
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const (
    apiURL   = "http://localhost:8081"
    apiToken = "your-secret-token"
)

type PromoteRequest struct {
    SkipAnalysis bool `json:"skipAnalysis"`
}

func listCanaries() ([]byte, error) {
    req, _ := http.NewRequest("GET", apiURL+"/api/v1/canaries", nil)
    req.Header.Set("Authorization", "Bearer "+apiToken)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    return buf.Bytes(), nil
}

func promoteCanary(namespace, name string, skipAnalysis bool) error {
    url := fmt.Sprintf("%s/api/v1/namespaces/%s/canaries/%s/promote", 
        apiURL, namespace, name)
    
    reqBody := PromoteRequest{SkipAnalysis: skipAnalysis}
    jsonData, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+apiToken)
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    _, err := client.Do(req)
    return err
}
```

## Deployment

When deploying Flagger with the API enabled, update your deployment manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flagger
  namespace: flagger-system
spec:
  template:
    spec:
      containers:
      - name: flagger
        image: ghcr.io/fluxcd/flagger:latest
        args:
        - -mesh-provider=istio
        - -metrics-server=http://prometheus:9090
        - -enable-api=true
        - -api-port=8081
        env:
        - name: API_TOKEN
          valueFrom:
            secretKeyRef:
              name: flagger-api-token
              key: token
        ports:
        - name: http
          containerPort: 8080
        - name: api
          containerPort: 8081
```

And create a Service to expose the API:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: flagger-api
  namespace: flagger-system
spec:
  type: ClusterIP
  ports:
  - name: api
    port: 8081
    targetPort: 8081
  selector:
    app: flagger
```
