# LGTM LBAC Proxy

**Label-Based Access Control for the LGTM Stack** (Loki, Grafana, Tempo, Mimir/Prometheus)

[![Go Report Card](https://goreportcard.com/badge/github.com/binhnguyenduc/lgtm-lbac-proxy)](https://goreportcard.com/report/github.com/binhnguyenduc/lgtm-lbac-proxy)
[![Build Status](https://img.shields.io/github/actions/workflow/status/binhnguyenduc/lgtm-lbac-proxy/release.yml?label=build)](https://github.com/binhnguyenduc/lgtm-lbac-proxy/actions/workflows/release.yml)
[![GoDoc](https://godoc.org/github.com/binhnguyenduc/lgtm-lbac-proxy?status.svg)](https://godoc.org/github.com/binhnguyenduc/lgtm-lbac-proxy)
[![Release](https://img.shields.io/github/v/release/binhnguyenduc/lgtm-lbac-proxy)](https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/latest)
![License](https://img.shields.io/github/license/binhnguyenduc/lgtm-lbac-proxy)

---

## 📢 Notice

### Origin and Credits

This project is **based on** [multena-proxy](https://github.com/gepaplexx/multena-proxy) originally developed by **Gepardec**.

We extend our sincere gratitude to the Gepardec team for creating the foundational multi-tenancy proxy architecture that made this project possible. The original work established excellent patterns for label-based access control in observability stacks.

**Original Repository**: https://github.com/gepaplexx/multena-proxy

### Differences from Original

This repository has evolved as an **independent project** with a specific focus on the LGTM stack:

| Aspect | Original (multena-proxy) | This Project (lgtm-lbac-proxy) |
|--------|--------------------------|--------------------------------|
| **Focus** | General multi-tenancy proxy | LGTM stack specialization |
| **Tempo Support** | ❌ Not implemented | ✅ Full TraceQL enforcement |
| **Branding** | Generic "multena" naming | LGTM-specific branding |
| **Repository** | Gepardec organization | Independent maintenance |
| **Development** | Upstream changes | LGTM-focused enhancements |
| **Target Users** | OpenShift/Thanos users | LGTM stack users |

**Key Enhancements:**
- ✅ **Grafana Tempo Support**: Full TraceQL query enforcement with tenant label injection
- ✅ **Complete LGTM Coverage**: Metrics (Prometheus/Mimir), Logs (Loki), and Traces (Tempo)
- ✅ **LGTM-Optimized**: Documentation, examples, and configurations tailored for LGTM deployments
- ✅ **Active Development**: Ongoing enhancements focused on LGTM stack integration

**Compatibility**: The core authorization logic, configuration format, and API remain compatible with the original design.

---

## Overview

**LGTM LBAC Proxy** is a multi-tenancy authorization proxy designed specifically for the LGTM observability stack. Built with Label-Based Access Control (LBAC) at its core, it ensures secure and granular access to your observability data.

The proxy intercepts queries to Prometheus/Thanos, Loki, and Tempo, validates JWT tokens, enforces tenant label restrictions, and forwards authorized queries to upstream servers.

### Key Features

| Feature | Description |
|---------|-------------|
| **LGTM Stack Coverage** | Complete support for Metrics (Prometheus/Thanos/Mimir), Logs (Loki), and Traces (Tempo) |
| **Label-Based Access Control** | Fine-grained access control based on tenant labels (namespace, team, environment, etc.) |
| **JWT Authentication** | Secure authentication via OAuth/OIDC with JWKS validation |
| **Flexible Label Stores** | ConfigMap or MySQL-based label storage |
| **Query Enforcement** | Automatic injection of tenant filters into PromQL, LogQL, and TraceQL queries |
| **Admin Bypass** | Optional admin group with cluster-wide access |
| **Multi-Tenant Support** | Single user can access multiple tenant namespaces |
| **Secure Communication** | mTLS support for upstream connections |

### Currently Supported

- ✅ **Metrics**: Prometheus, Thanos, Mimir (PromQL enforcement)
- ✅ **Logging**: Loki (LogQL enforcement)
- ✅ **Traces**: Tempo (TraceQL enforcement) 🆕
- ⏳ **Profiles**: Planned for future release

---

## How It Works

```mermaid
graph TD
    A[Receive Request] -->|Extract JWT Token| B{Validate Token?}
    B -->|Invalid| E[403 Forbidden]
    B -->|Valid| D{Validate Labels?}
    D -->|No Labels| E
    D -->|Has #cluster-wide| F[Forward to Upstream]
    D -->|Has Labels| G{Enforce Query?}
    G -->|Unauthorized Label| E
    G -->|Success| H[Inject Tenant Filter]
    H --> F
    F --> I[Stream Response]
```

**Authorization Flow:**

1. **Request Reception**: Proxy receives a query request (PromQL, LogQL, or TraceQL)
2. **JWT Validation**: Extracts and validates JWT token from Authorization header
3. **Label Retrieval**: Fetches allowed tenant labels for user/groups from label store
4. **Admin Bypass Check**: Users with `#cluster-wide` label skip enforcement
5. **Query Parsing**: Parses query using appropriate parser (PromQL/LogQL/TraceQL)
6. **Label Validation**: Checks if existing tenant labels in query are authorized
7. **Label Injection**: Injects tenant label filters if missing (e.g., `{namespace="prod"}`)
8. **Proxy Forward**: Forwards modified query to upstream (Prometheus/Loki/Tempo)
9. **Response Stream**: Streams response back to client

---

## Quick Start

### Binary

```bash
# Download latest release
wget https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/latest/download/lgtm-lbac-proxy

# Make executable
chmod +x lgtm-lbac-proxy

# Run with config
./lgtm-lbac-proxy
```

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/configs:/etc/config/config:ro \
  ghcr.io/binhnguyenduc/lgtm-lbac-proxy:latest
```

### Helm

```bash
helm install lgtm-lbac-proxy ./helm/lgtm-lbac-proxy \
  --set proxy.thanos.url=https://thanos-query:9091 \
  --set proxy.loki.url=https://loki-query:3100 \
  --set proxy.tempo.url=https://tempo-query:3100
```

---

## Configuration

### Basic Configuration

Create `config.yaml`:

```yaml
web:
  jwks_cert_url: "https://your-oauth-provider.com/certs"
  label_store_kind: configmap  # or "mysql"
  oauth_group_name: "groups"
  auth_header: "Authorization"  # Configurable auth header name

admin:
  bypass: true
  group: "cluster-admins"

thanos:
  url: "https://thanos-querier:9091"
  tenant_label: "namespace"
  use_mutual_tls: false

loki:
  url: "https://loki-query-frontend:3100"
  tenant_label: "kubernetes_namespace_name"
  actor_header: "X-Loki-Actor"

tempo:
  url: "https://tempo-query-frontend:3100"
  tenant_label: "resource.namespace"
  actor_header: "X-Tempo-User"
```

### Label Configuration

Create `labels.yaml` (ConfigMap mode):

```yaml
# User-specific labels
alice@example.com:
  prod: true
  staging: true

bob@example.com:
  dev: true

# Group-based labels
engineering-team:
  prod: true
  staging: true
  dev: true

# Admin access
cluster-admins:
  "#cluster-wide": true
```

---

## Query Examples

### Metrics (PromQL)

**Before (Original Query)**:
```promql
rate(http_requests_total[5m])
```

**After (Enforced)**:
```promql
rate(http_requests_total{namespace="prod"}[5m])
```

### Logs (LogQL)

**Before (Original Query)**:
```logql
{app="nginx"} |= "error"
```

**After (Enforced)**:
```logql
{kubernetes_namespace_name="prod",app="nginx"} |= "error"
```

### Traces (TraceQL) 🆕

**Before (Original Query)**:
```traceql
{ span.http.status_code >= 500 }
```

**After (Enforced)**:
```traceql
{ resource.namespace="prod" && span.http.status_code >= 500 }
```

---

## Datasource Configuration

### Grafana Configuration

Configure Grafana datasources to point to the proxy:

**Prometheus/Thanos:**
```
URL: http://lgtm-lbac-proxy:8080/api/v1
```

**Loki:**
```
URL: http://lgtm-lbac-proxy:8080/loki/api/v1
```

**Tempo:**
```
URL: http://lgtm-lbac-proxy:8080/tempo/api
```

All requests must include valid JWT token in Authorization header.

---

## Building from Source

```bash
# Clone repository
git clone https://github.com/binhnguyenduc/lgtm-lbac-proxy.git
cd lgtm-lbac-proxy

# Build binary
make build

# Run tests
make test

# Build Docker image
make docker-build-full
```

See [BUILD.md](BUILD.md) for detailed build instructions.

---

## Migration from multena-proxy

If you're migrating from the original multena-proxy, see [MIGRATION.md](MIGRATION.md) for a comprehensive guide.

**Key Changes:**
- Binary name: `multena-proxy` → `lgtm-lbac-proxy`
- Docker image: `ghcr.io/gepaplexx/multena-proxy` → `ghcr.io/binhnguyenduc/lgtm-lbac-proxy`
- Helm chart: `gp-multena` → `lgtm-lbac-proxy`

**No breaking changes** to configuration format or API endpoints.

---

## Documentation

- [Configuration Guide](configs/config.yaml)
- [Build Instructions](BUILD.md)
- [Migration Guide](MIGRATION.md)
- [Helm Chart README](helm/README.md)

---

## Contributing

Contributions are welcome! This is an independent project focused on LGTM stack support.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

---

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- **Original Project**: [multena-proxy](https://github.com/gepaplexx/multena-proxy) by Gepardec
- **Contributors**: Thank you to everyone who has contributed to making LGTM stack observability more secure

---

## Support

- **Issues**: [GitHub Issues](https://github.com/binhnguyenduc/lgtm-lbac-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/binhnguyenduc/lgtm-lbac-proxy/discussions)
- **Documentation**: [GitHub Wiki](https://github.com/binhnguyenduc/lgtm-lbac-proxy/wiki)
