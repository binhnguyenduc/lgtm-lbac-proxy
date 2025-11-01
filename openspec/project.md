# Project Context

## Purpose

**Multena Proxy** is a multi-tenancy authorization proxy for the LGTM stack (Loki, Grafana, Tempo, Mimir/Prometheus). It implements Label-Based Access Control (LBAC) to enforce tenant isolation on observability queries.

### Core Functionality
- Intercepts queries to Prometheus/Thanos and Loki
- Validates JWT tokens for authentication
- Enforces tenant label restrictions based on user/group permissions
- Forwards authorized queries to upstream observability systems
- Ensures users can only access data they are authorized to view

### Key Value Proposition
Making the LGTM-Stack **mul**ti **tena**ncy ready by providing secure and granular user authorization based on assigned tenant labels.

## Tech Stack

### Language & Runtime
- **Go 1.23.4** - Primary programming language
- Modular architecture with builder pattern initialization

### Core Dependencies
- **github.com/MicahParks/keyfunc/v3** (v3.3.5) - JWKS key management with auto-refresh
- **github.com/golang-jwt/jwt/v5** (v5.2.1) - JWT parsing and validation
- **github.com/gorilla/mux** (v1.8.1) - HTTP routing
- **github.com/spf13/viper** (v1.19.0) - Configuration management with hot-reload
- **github.com/rs/zerolog** (v1.33.0) - Structured logging
- **github.com/prometheus/client_golang** (v1.20.5) - Metrics exposition
- **github.com/slok/go-http-metrics** (v0.13.0) - HTTP metrics middleware

### Query Enforcement Libraries
- **github.com/prometheus-community/prom-label-proxy** (v0.11.0) - PromQL label injection
- **github.com/observatorium/api** (v0.1.3) - LogQL parsing and enforcement
- **github.com/prometheus/prometheus** (v0.55.1) - PromQL parser

### Database & Storage
- **github.com/go-sql-driver/mysql** (v1.8.1) - MySQL driver for label store
- **github.com/fsnotify/fsnotify** (v1.8.0) - File system notifications for ConfigMap watching

### Testing
- **github.com/stretchr/testify** (v1.10.0) - Assertion and mocking framework

### Deployment
- Kubernetes via Helm chart (gp-helm-charts/gp-multena)
- Supports both ConfigMap and MySQL label store providers

## Project Conventions

### Code Style

**Naming Conventions:**
- **Package names**: lowercase, single word (e.g., `main`, `auth`, `config`)
- **File names**: lowercase with underscores for separation (e.g., `auth.go`, `enforcer_promql.go`, `labelstore_test.go`)
- **Functions/Methods**: camelCase with capital first letter for exported (e.g., `ValidateToken`, `EnforceGet`)
- **Variables**: camelCase (e.g., `userName`, `allowedLabels`)
- **Constants**: UPPER_CASE with underscores (e.g., `DEFAULT_PORT`)
- **Structs**: PascalCase (e.g., `App`, `PromQLEnforcer`, `MySQLHandler`)

**Go Conventions:**
- Follow standard Go formatting (use `gofmt`)
- Exported identifiers start with capital letter
- Unexported identifiers start with lowercase letter
- Interface names end with `-er` suffix (e.g., `Handler`, `Enforcer`)
- Error handling: explicit error returns, no panic in library code
- Logging: use zerolog for structured logging with appropriate log levels

**File Organization:**
- Test files alongside implementation: `auth.go` → `auth_test.go`
- Configuration files in `configs/` directory
- All source files at root level (flat structure)
- No nested package hierarchy

### Architecture Patterns

**Builder Pattern:**
- `App` struct initialization uses builder pattern:
  ```
  App → WithConfig() → WithSAT() → WithTLSConfig() →
  WithJWKS() → WithLabelStore() → WithHealthz() →
  WithRoutes() → StartServer()
  ```

**Handler Pattern:**
- All HTTP routes use unified handler pattern:
  ```
  Request → Extract JWT → Validate Token → Retrieve Labels →
  Enforce Query → Proxy to Upstream → Stream Response
  ```

**Strategy Pattern:**
- `LabelStore` interface with multiple implementations:
  - `ConfigMapHandler` - reads from YAML ConfigMap
  - `MySQLHandler` - queries database dynamically

**Enforcer Pattern:**
- Separate enforcers for different query languages:
  - `PromQLEnforcer` - Prometheus query enforcement
  - `LogQLEnforcer` - Loki query enforcement

**Key Architectural Principles:**
- **Single Responsibility**: Each component handles one concern (auth, enforcement, routing, storage)
- **Interface Segregation**: Small, focused interfaces (e.g., `LabelStore`)
- **Dependency Injection**: Dependencies passed via builder pattern
- **Separation of Concerns**: Clear boundaries between auth, enforcement, and proxying

### Testing Strategy

**Test Organization:**
- Unit tests for each component: `auth_test.go`, `enforcer_promql_test.go`, `enforcer_logql_test.go`
- Integration tests in `main_test.go` simulate full request flow
- Test files co-located with implementation files

**Testing Patterns:**
- **Table-driven tests**: Multiple scenarios tested with structured test cases
- **Mock objects**: Use testify mocks for external dependencies
- **Test fixtures**: Sample JWTs, queries, and configurations in test files

**Test Coverage:**
```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -v -run TestEnforceGet ./...
```

**Test Focus Areas:**
- Token parsing and JWKS validation
- Query enforcement with various PromQL/LogQL patterns
- Label store implementations (ConfigMap and MySQL)
- HTTP routing and handler logic
- Error handling and edge cases

### Git Workflow

**Branching Strategy:**
- `main` - production-ready code
- Feature branches: `feature/{description}`
- Bug fixes: `bugfix/{description}`
- Hotfixes: `hotfix/{description}`

**Commit Conventions:**
- Use clear, descriptive commit messages
- Format: Subject line + blank line + detailed description (if needed)
- Professional format WITHOUT Claude Code footer
- Subject line: imperative mood, capitalized, no period
- Examples:
  - "Add MySQL label store provider"
  - "Fix JWT validation for missing claims"
  - "Refactor enforcer to support multiple label values"

**CI/CD:**
- GitHub Actions workflows for build and release
- Automated testing on pull requests
- Release workflow publishes container images

## Domain Context

### Multi-Tenancy & Label-Based Access Control (LBAC)

**Tenant Isolation:**
- Each tenant identified by a label (typically `namespace` in Kubernetes)
- Users/groups assigned specific tenant label values
- Queries automatically scoped to user's allowed tenant labels

**Special Label:**
- `#cluster-wide` - grants skip enforcement (admin access)
- Users with this label bypass all enforcement steps

**Label Enforcement:**
- Validates existing label matchers in queries against allowed labels
- Injects label matchers if missing from query
- Constructs minimal queries for empty queries: `{namespace="ns1"}`
- Supports multiple tenant values: `{namespace=~"ns1|ns2|ns3"}`

### Authentication & Authorization Flow

1. **Extract Token**: Get JWT from `Authorization: Bearer <token>` header
2. **Validate Token**: Verify signature using JWKS from OAuth provider
3. **Extract Claims**: Get `preferred_username`, `email`, and groups
4. **Retrieve Labels**: Lookup allowed tenant labels from label store
5. **Admin Check**: If user in admin group and bypass enabled, skip enforcement
6. **Enforce Query**: Modify query to inject/validate tenant label matchers
7. **Proxy Request**: Forward to upstream (Thanos/Loki) with mTLS if configured
8. **Stream Response**: Return results to client

### Supported Observability Systems

**Currently Supported:**
- ✅ Metrics (Prometheus/Thanos/Mimir)
- ✅ Logging (Loki)
- ❌ Traces (Tempo) - planned
- ❌ Profiles - planned

**Integration Points:**
- **Prometheus/Thanos Routes**: `/api/v1/query`, `/api/v1/query_range`, `/api/v1/series`, `/api/v1/labels`, `/api/v1/label/{name}/values`
- **Loki Routes**: `/loki/api/v1/query`, `/loki/api/v1/query_range`, `/loki/api/v1/series`, `/loki/api/v1/labels`, `/loki/api/v1/label/{name}/values`

### Query Language Specifics

**PromQL Enforcement:**
- Uses `prometheus/prometheus` parser
- Validates existing label matchers against allowed labels
- Injects label matcher if missing
- Example: `up` → `up{namespace="ns1"}`

**LogQL Enforcement:**
- Uses `observatorium/api` logql parser
- Walks AST to find `StreamMatcherExpr`
- Validates/injects tenant label matchers
- Example: `{job="app"}` → `{job="app", kubernetes_namespace_name="ns1"}`

## Important Constraints

### Security Requirements

**Authentication:**
- MUST validate JWT tokens using JWKS from OAuth provider
- MUST support JWKS auto-refresh for key rotation
- MUST extract and validate user/group claims

**Authorization:**
- MUST enforce tenant label restrictions on all queries
- MUST NOT allow users to access data outside their assigned labels
- MUST validate label matchers in queries against allowed labels

**Transport Security:**
- SHOULD use TLS for all external communication
- SHOULD support mTLS for upstream connections
- CAN skip TLS verification only in development (`tls_verify_skip: true`)

**Sensitive Data:**
- MUST NOT log JWT tokens unless explicitly enabled for debugging
- MUST use appropriate log levels (trace exposes sensitive data)

### Operational Constraints

**Configuration Hot-Reload:**
- Configuration changes MUST be picked up without restart (via fsnotify)
- Label ConfigMap changes MUST be applied dynamically

**Performance:**
- Label lookup SHOULD be fast (in-memory for ConfigMap)
- Database queries SHOULD be cached to reduce load
- Query enforcement MUST NOT significantly impact latency

**High Availability:**
- Stateless design allows horizontal scaling
- No local state dependency (except ConfigMap/DB)

**Kubernetes Integration:**
- MUST read ServiceAccount token from `/var/run/secrets/kubernetes.io/serviceaccount/token`
- MUST support ConfigMap-based configuration
- SHOULD integrate with Grafana Operator datasources

### Development Constraints

**Local Development:**
- Set `dev.enabled: true` for local development
- Provide `web.service_account_token` to avoid K8s dependency
- Use `./configs/` directory for configuration files

**Testing:**
- Unit tests MUST NOT depend on external services
- Integration tests SHOULD use mock servers
- Test coverage SHOULD be maintained for critical paths

**Compatibility:**
- MUST support Prometheus-style API (Thanos, Mimir)
- MUST support Loki query API
- SHOULD be compatible with standard OAuth2/OIDC providers

## External Dependencies

### OAuth/OIDC Provider
- **Purpose**: JWT token issuance and JWKS endpoint
- **Required**: JWKS certificate URL (`web.jwks_cert_url`)
- **Claims**: `preferred_username`, `email`, configurable group claim
- **Integration**: Token validation via JWKS, auto-refresh support

### Upstream Observability Systems

**Thanos/Prometheus:**
- **URL**: Configured in `thanos.url`
- **Authentication**: Optional mTLS (`thanos.cert`, `thanos.key`)
- **Headers**: Optional custom headers (e.g., `X-Scope-OrgID`)
- **Label**: Tenant label to enforce (e.g., `namespace`)

**Loki:**
- **URL**: Configured in `loki.url`
- **Authentication**: Optional mTLS (`loki.cert`, `loki.key`)
- **Headers**: Optional custom headers (e.g., `X-Scope-OrgID`)
- **Label**: Tenant label to enforce (e.g., `kubernetes_namespace_name`)
- **Actor Header**: Optional base64-encoded username/email for fair usage tracking

### Label Store Providers

**ConfigMap (Recommended for single cluster):**
- **Source**: Kubernetes ConfigMap with `labels.yaml`
- **Format**: YAML with user/group → label mappings
- **Watching**: fsnotify-based hot-reload
- **Tool**: Use `multena-rbac-collector` for automated label collection

**MySQL (For centralized multi-cluster):**
- **Connection**: `db.host`, `db.port`, `db.user`, `db.password_path`
- **Query**: Custom SQL query to retrieve labels
- **Token Key**: JWT claim to use in query (`email`, `username`, or `groups`)
- **Caching**: Recommended to reduce database load

### Grafana Integration

**Datasource Configuration:**
- Grafana connects to Multena as Prometheus/Loki datasource
- OAuth token forwarded in `Authorization` header
- Supports Grafana Operator datasources for automated setup

**Alerting Support:**
- Fallback header for Grafana alerting (no user token)
- Separate JWKS certificate for alert service account
- Configured via `alert.token_header` and `alert.alert_cert_url`

### Certificate Authorities

**System CAs:**
- Loaded from `/etc/ssl/ca/ca-certificates.crt`

**Additional CAs:**
- Configurable via `web.trusted_root_ca_path`
- Used for validating upstream TLS certificates

### Metrics & Monitoring

**Prometheus Metrics:**
- Exposed on port `8081` (configurable via `web.metrics_port`)
- HTTP request metrics via `go-http-metrics` middleware
- Go runtime metrics via `prometheus/client_golang`

### Helm Chart Repository
- **Repository**: https://gepaplexx.github.io/gp-helm-charts/
- **Chart**: `gepardec/gp-multena`
- **Deployment**: Kubernetes namespace (typically Grafana namespace)
- **Dependencies**: Optional `grafana-operator-datasources`, `multena-rbac-collector`
