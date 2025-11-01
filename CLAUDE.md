<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Multena Proxy** is a multi-tenancy authorization proxy for the LGTM stack (Loki, Grafana, Tempo, Mimir/Prometheus). It implements Label-Based Access Control (LBAC) to enforce tenant isolation on observability queries.

**Core Purpose**: Intercepts queries to Prometheus/Thanos and Loki, validates JWT tokens, enforces tenant label restrictions, and forwards authorized queries to upstream servers.

## Architecture

### Request Flow

1. **Authentication** (`auth.go`): Extract and validate JWT token from Authorization header
2. **Label Validation** (`labelstore.go`): Retrieve allowed tenant labels for user/group from ConfigMap or MySQL
3. **Query Enforcement** (`enforce.go`, `enforcer_promql.go`, `enforcer_logql.go`): Modify query to enforce tenant label restrictions
4. **Proxy** (`routes.go`): Forward request to upstream (Thanos/Loki) with injected labels

### Key Components

**Main Application** (`main.go`):
- `App` struct holds application state: JWKS, config, TLS, label store, routers
- Builder pattern initialization: `WithConfig()` → `WithSAT()` → `WithTLSConfig()` → `WithJWKS()` → `WithLabelStore()` → `WithHealthz()` → `WithRoutes()` → `StartServer()`
- Runs two HTTP servers: metrics (8081) and proxy (8080)

**Configuration** (`config.go`):
- Viper-based configuration with hot-reload via fsnotify
- Config paths: `/etc/config/config/` (Kubernetes) or `./configs` (local dev)
- Supports ConfigMap and MySQL label stores, mTLS, OAuth settings

**Label Stores** (`labelstore.go`):
- `ConfigMapHandler`: Reads labels from `labels.yaml`, watches for changes
- `MySQLHandler`: Queries database for user/group labels dynamically
- Special label `#cluster-wide` grants skip enforcement (admin access)

**Query Enforcers**:
- `PromQLEnforcer` (`enforcer_promql.go`): Uses prometheus/prometheus parser and prom-label-proxy to inject label matchers
- `LogQLEnforcer` (`enforcer_logql.go`): Uses observatorium/api logql parser to inject label matchers
- Both ensure queries only access data matching user's allowed tenant labels

**Routing** (`routes.go`):
- Loki routes: `/loki/api/v1/query`, `/loki/api/v1/query_range`, etc.
- Thanos/Prometheus routes: `/api/v1/query`, `/api/v1/query_range`, etc.
- Each route uses `handler()` to orchestrate auth → validate → enforce → proxy

**Authentication** (`auth.go`):
- JWT parsing with JWKS validation via MicahParks/keyfunc
- Extracts `preferred_username`, `email`, and configurable group claim
- Admin bypass: users in admin group skip label enforcement if enabled
- Alert support: fallback header for Grafana alerting (no user token)

## Development Commands

### Build
```bash
go build -o multena-proxy .
```

### Run Locally
```bash
# Requires config.yaml and labels.yaml in ./configs/
go run .
```

### Testing

**Run all tests**:
```bash
go test -v ./...
```

**Run tests with coverage**:
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Run single test**:
```bash
# Example: test a specific function
go test -v -run TestEnforceGet ./...
go test -v -run TestPromQLEnforcer ./...
```

**Test files organized by component**:
- `auth_test.go`: Token parsing and validation
- `enforcer_promql_test.go`: PromQL query enforcement
- `enforcer_logql_test.go`: LogQL query enforcement
- `labelstore_test.go`: Label store implementations
- `routes_test.go`: HTTP routing and handlers
- `main_test.go`: Integration tests

## Configuration

### Required Files

**`configs/config.yaml`** - Main configuration:
- `web.jwks_cert_url`: OAuth provider JWKS endpoint (required)
- `web.oauth_group_name`: JWT claim containing groups (default: "groups")
- `web.auth_header`: HTTP header name for JWT authentication (default: "Authorization")
- `web.label_store_kind`: "configmap" or "mysql"
- `thanos.url`: Prometheus/Thanos endpoint
- `thanos.tenant_label`: Label to enforce (e.g., "namespace")
- `loki.url`: Loki endpoint
- `loki.tenant_label`: Label to enforce (e.g., "namespace")
- `admin.bypass`: Enable admin group bypass
- `admin.group`: Admin group name

**`configs/labels.yaml`** - User/group label mappings (ConfigMap mode):
```yaml
username1:
  namespace1: true
  namespace2: true
group1:
  '#cluster-wide': true  # Grants cluster-wide access
```

### Local Development

Set `dev.enabled: true` in config.yaml and provide `web.service_account_token` to avoid reading from `/var/run/secrets/kubernetes.io/serviceaccount/token`.

## Important Patterns

### Query Enforcement Logic

**PromQL**: Parses query with prometheus parser, validates existing label matchers against allowed labels, injects label matcher if missing.

**LogQL**: Parses query with logql parser, walks AST to find `StreamMatcherExpr`, validates/injects tenant label matchers.

**Empty queries**: Both enforcers construct minimal query like `{namespace="ns1"}` or `{namespace=~"ns1|ns2"}` for multiple labels.

### Error Handling

All handler errors return HTTP 403 Forbidden with logged error message. Critical errors (config load, JWKS fetch, DB connection) cause fatal exit with log.Fatal().

### TLS Configuration

- Loads system CA certificates from `/etc/ssl/ca/ca-certificates.crt`
- Optionally loads additional CAs from `web.trusted_root_ca_path`
- Supports mTLS with `thanos.cert/key` and `loki.cert/key`
- Can skip TLS verification with `web.tls_verify_skip: true` (insecure, dev only)

### Actor Header for Loki Fair Usage

If `loki.actor_header` or `thanos.actor_header` is set, the proxy injects a base64-encoded username/email into the specified header for fair usage tracking by upstream systems.

## Testing Strategy

- Unit tests for each enforcer with various query patterns
- Mock label stores for auth/validation tests
- Integration tests in `main_test.go` simulate full request flow
- Use table-driven tests for multiple query scenarios

## Dependencies

Key external libraries:
- `github.com/MicahParks/keyfunc/v3`: JWKS key management with auto-refresh
- `github.com/prometheus-community/prom-label-proxy`: PromQL label injection
- `github.com/observatorium/api`: LogQL parsing
- `github.com/gorilla/mux`: HTTP routing
- `github.com/spf13/viper`: Configuration management with hot-reload
- `github.com/rs/zerolog`: Structured logging
- `github.com/golang-jwt/jwt/v5`: JWT parsing and validation
