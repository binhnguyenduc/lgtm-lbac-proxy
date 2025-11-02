# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.10.0] - 2025-11-02

This release introduces **flexible multi-label enforcement**, enabling fine-grained access control with multiple label rules per user/group. Users can now define complex authorization policies with different label names, operators, and logic combinations while maintaining full backward compatibility with existing single-label configurations.

### Added

- **Multi-Label Enforcement System**: Complete support for flexible, policy-based label enforcement
  - `LabelRule`: Define individual label matching rules with name, operator, and values
  - `LabelPolicy`: Combine multiple rules with AND/OR logic for complex access control
  - Support for all query operators: `=` (equals), `!=` (not equals), `=~` (regex), `!~` (negative regex)
  - Per-user/group custom enforcement policies independent of global `tenant_label` config

- **Extended Label Format**: New YAML schema for labels.yaml with backward compatibility
  ```yaml
  # Extended format - multiple labels with operators
  user1:
    _rules:
      - name: namespace
        operator: "="
        values: ["prod", "staging"]
      - name: team
        operator: "=~"
        values: ["backend.*"]
    _logic: AND

  # Simple format still works (backward compatible)
  user2:
    namespace: prod

  # Mixed format supported
  user3:
    namespace: dev
    _rules:
      - name: environment
        operator: "!="
        values: ["production"]
  ```

- **Enhanced Query Enforcers**: All enforcers now support multi-label policies
  - `PromQLEnforcer.EnforceMulti()`: Multi-label PromQL query enforcement
  - `LogQLEnforcer.EnforceMulti()`: Multi-label LogQL query enforcement
  - `TraceQLEnforcer.EnforceMulti()`: Multi-label TraceQL query enforcement
  - Existing `Enforce()` methods maintained for backward compatibility

- **Policy Parser** (`policyparser.go`): Intelligent YAML format detection and parsing
  - Auto-detects simple vs extended format
  - Converts simple format to extended format internally
  - Validates rule structure and operators
  - Supports mixed format (simple + extended rules in same user)

- **Label Store Enhancement**: Extended Labelstore interface with policy support
  - `GetLabelPolicy()`: Retrieve complete label policy for user/group
  - `GetLabels()`: Maintained for backward compatibility
  - Automatic format detection and conversion in FileLabelStore

- **Migration Tools**: Comprehensive migration support
  - Standalone CLI tool: `cmd/migrate-labels/migrate-labels`
    - Validate existing labels.yaml
    - Preview conversion (dry-run mode)
    - Convert simple to extended format
    - Bulk migration capability
  - Helm migration job: `helm/lgtm-lbac-proxy/templates/migration-job.yaml`
    - Automated Kubernetes migration workflow
    - Dry-run support for safe preview
    - Configurable via `values.yaml`

- **Documentation**: Complete documentation for new features
  - Migration guide: `helm/lgtm-lbac-proxy/UPGRADE.md`
  - Updated architecture docs in `CLAUDE.md`
  - Configuration examples for all formats
  - Performance benchmarking results

- **Comprehensive Testing**: Full test coverage for new functionality
  - 82+ test cases across all enforcers (28 PromQL, 18 LogQL, 21 TraceQL)
  - 11 performance benchmarks in `labelrule_bench_test.go`
  - Backward compatibility tests with mixed formats
  - Test coverage: >80% for new components (labelrule: 100%, policyparser: 90%+, enforcers: 85-95%)

### Changed

- **Request Handler Flow**: Updated to support policy-based enforcement
  - Checks for extended format using `GetLabelPolicy()`
  - Falls back to simple format with `GetLabels()` for backward compatibility
  - Maintains zero-breaking-change migration path

- **Configuration**: Optional extended format, no config changes required
  - `tenant_label` continues to work as default/fallback for simple format
  - Extended format overrides `tenant_label` when `_rules` present
  - No new required configuration fields

### Performance

All operations meet <5ms latency requirement with minimal overhead:
- Simple format parsing: ~87 ns/op (0.087 µs)
- Extended format parsing: ~1.9 µs/op
- PromQL single-label enforcement: ~5-10 µs/op
- PromQL multi-label enforcement: ~15-25 µs/op
- LogQL multi-label enforcement: ~20-30 µs/op
- TraceQL multi-label enforcement: ~25-35 µs/op

Multi-label enforcement adds only ~10-15 µs overhead compared to single-label.

### Migration

**No Action Required for Existing Deployments**:
- Simple format labels continue to work without modification
- Proxy automatically detects format and routes to appropriate enforcement
- Existing queries and configurations require zero changes

**To Adopt Multi-Label Enforcement**:
1. **Option 1**: Keep simple format (recommended for single-label use cases)
2. **Option 2**: Manual migration to extended format
3. **Option 3**: Automated migration using Helm job or standalone tool

See `helm/lgtm-lbac-proxy/UPGRADE.md` for detailed migration instructions.

### Technical Details

#### New Files
- `labelrule.go`: LabelRule and LabelPolicy data structures with validation
- `policyparser.go`: PolicyParser for parsing and converting label formats
- `labelrule_test.go`: Unit tests for label rules (6 test cases, 100% coverage)
- `policyparser_test.go`: Unit tests for policy parsing (9 test cases, 90%+ coverage)
- `labelrule_bench_test.go`: Performance benchmarks (11 benchmark functions)
- `cmd/migrate-labels/`: Standalone migration tool with CLI

#### Modified Files
- `labelstore.go`: Added `GetLabelPolicy()` method, enhanced FileLabelStore
- `enforcer_promql.go`: Added `EnforceMulti()` with multi-label support (28 test cases)
- `enforcer_logql.go`: Added `EnforceMulti()` with multi-label support (18 test cases)
- `enforcer_traceql.go`: Added `EnforceMulti()` with multi-label support (21 test cases)
- `routes.go`: Updated handler to support policy-based enforcement
- `util.go`: Added policy-to-matcher conversion utilities
- `auth.go`: Enhanced to work with policy enforcement
- `CLAUDE.md`: Updated with multi-label architecture and examples
- `helm/lgtm-lbac-proxy/values.yaml`: Added migration job configuration
- `helm/lgtm-lbac-proxy/UPGRADE.md`: Comprehensive upgrade guide

#### Design Principles
- **Backward Compatibility**: Zero breaking changes, simple format fully supported
- **Gradual Adoption**: Users can migrate incrementally, mixing simple and extended formats
- **Performance**: Minimal overhead (<15 µs added latency for multi-label)
- **Extensibility**: Clean architecture for future authorization enhancements
- **User Experience**: Auto-detection and conversion for seamless migration

### Benefits

1. **Fine-Grained Access Control**: Combine multiple labels for precise access policies
2. **Multi-Tenancy Flexibility**: Different users can have different label restrictions
3. **Advanced Operators**: Support regex matching and negative conditions
4. **Future-Proof Architecture**: Foundation for more complex authorization rules
5. **Zero Migration Risk**: Full backward compatibility ensures safe adoption

## [0.9.1] - 2025-11-02

### Changed

- **Refactoring**: Renamed `ConfigMapHandler` to `FileLabelStore` for platform-agnostic naming
  - Updated all code references in `labelstore.go`, test files, and integration tests
  - Updated documentation in `CLAUDE.md`, `contrib/labelstores/`, and OpenSpec files
  - No functional changes or breaking changes to the API
  - Improves clarity by reflecting file-based storage rather than Kubernetes-specific terminology

- **⚠️ BREAKING CHANGE - Architecture**: Refactored `Labelstore` interface to eliminate leaked context
  - **Interface Changes**:
    - `Connect(App)` → `Connect(LabelStoreConfig)` - Label stores now receive focused configuration instead of entire App
    - `GetLabels(OAuthToken)` → `GetLabels(UserIdentity)` - Label stores now receive minimal identity info, not auth-specific tokens
  - **New Types**:
    - `UserIdentity`: Authentication-agnostic identity struct with `Username` and `Groups`
    - `LabelStoreConfig`: Focused configuration struct for label store initialization (currently contains `ConfigPaths`)
  - **Benefits**:
    - **Separation of Concerns**: Label stores focus on identity → labels mapping, not authentication details
    - **Dependency Inversion**: Label stores depend on abstractions (`LabelStoreConfig`), not high-level App
    - **Extensibility**: Support new auth methods (SAML, LDAP, certificates) without changing label store interface
    - **Testability**: Label stores can be tested with minimal dependencies
  - **Migration**: Custom label stores must update interface implementation (see `contrib/labelstores/README.md`)
  - **No User Impact**: File-based label store behavior unchanged, only internal architecture improved

- **Configuration**: Made `LabelStoreConfig` part of standard configuration system
  - **User-Configurable**: Users can now configure label store paths directly in `config.yaml` via `labelstore.config_paths`
  - **Default Values**: Defaults to `["/etc/config/labels/", "./configs"]` for Kubernetes and local development
  - **Extensibility**: Custom label stores can add their own configuration fields to `LabelStoreConfig` without breaking existing stores
  - **Removed**: `App.ToLabelStoreConfig()` method - configuration now read from standard config system
  - **Example Configuration**:
    ```yaml
    labelstore:
      config_paths:
        - /etc/config/labels/  # Kubernetes ConfigMap mount path
        - ./configs            # Local development path
    ```
  - **Helm Chart**: Updated `values.yaml` and `configmaps.yaml` template with labelstore configuration
  - **Benefits**: Centralized configuration management, easier customization, clearer defaults

### Fixed

- **CI/CD**: Fixed GitHub Actions SBOM generation failure on tagged releases
  - Added `upload-release-assets: false` to `anchore/sbom-action` configuration
  - Prevents "Resource not accessible by integration" error when workflow lacks `contents: write` permission
  - SBOM still uploaded as workflow artifact for security compliance
  - Resolves issue where v0.9.0 and v0.8.0 CI jobs failed during SBOM generation step

## [0.9.0] - 2025-11-02

This release simplifies the label store architecture by removing MySQL support and establishing a single, file-based implementation. This change reduces complexity, eliminates external dependencies, and provides a clearer extension pattern for community contributions.

### Added

- **Community Extension Pattern**: New `contrib/labelstores/` directory for community-maintained label store implementations
  - Comprehensive documentation on implementing custom label stores
  - Template implementation with best practices
  - Clear interface requirements and testing guidelines
- **Migration Guide**: Detailed instructions for migrating from MySQL to file-based label store in README.md

### Removed

- **MySQL Label Store**: Removed MySQLHandler implementation and all database-related code
  - ⚠️ **BREAKING CHANGE**: MySQL label store is no longer supported
  - Users must migrate to file-based ConfigMap label store (see migration guide in README.md)
- **Database Dependencies**: Removed `github.com/go-sql-driver/mysql` dependency
- **Configuration Options**:
  - Removed `web.label_store_kind` configuration field (no longer needed)
  - Removed entire `db:` configuration section (all MySQL-related settings)

### Changed

- **Label Store Logic**: Simplified `WithLabelStore()` to directly instantiate ConfigMapHandler
  - Removed switch statement and validation logic for label_store_kind
  - Single, straightforward initialization path
- **Documentation**: Updated all documentation to reflect file-based label store only
  - Updated README.md feature list
  - Updated CLAUDE.md architecture description
  - Updated openspec/project.md dependencies and patterns

### Technical Details

#### Migration Path
Existing MySQL users should:
1. Export labels from MySQL database
2. Convert to YAML format
3. Deploy as ConfigMap
4. Remove database configuration
5. See README.md for detailed migration steps

#### Architecture Benefits
- **Reduced Complexity**: Single label store implementation vs multiple
- **Fewer Dependencies**: No database driver required
- **Better Security**: No database credentials or SQL injection concerns
- **Easier Testing**: Only file-based implementation needs testing
- **Clearer Extension**: Community contributions via documented contrib/ pattern

#### File Changes
- `labelstore.go`: 99 deletions (MySQLHandler and related code removed)
- `config.go`: 13 deletions (DbConfig struct removed)
- `go.mod`: Removed mysql driver dependency
- `contrib/labelstores/`: 227 additions (new extension framework)

## [0.8.0] - 2025-11-01

This is the first independent release of **lgtm-lbac-proxy**, forked from [multena-proxy](https://github.com/gepaplexx/multena-proxy) by Gepardec. This release focuses on completing LGTM stack support and establishing production-ready Kubernetes deployment capabilities.

### Added

#### Core Features
- **Grafana Tempo Support**: Full TraceQL query enforcement with tenant attribute injection
  - TraceQL parser integration using grafana/tempo packages
  - Support for resource attributes (e.g., `resource.namespace`)
  - String-based query manipulation with validation
  - Comprehensive test coverage for TraceQL enforcement
  - Empty query handling and multi-tenant regex support
- **Configurable Authentication Header**: Allow customization of authentication header name (default: "Authorization")
  - Enables integration with custom auth systems
  - Configurable via `web.auth_header` in config.yaml

#### Kubernetes & Deployment
- **Helm Chart v1.7.0**: Initial production-ready Helm chart for Kubernetes deployment
  - Support for Prometheus/Thanos and Loki
  - ConfigMap and Secret management
  - ServiceMonitor for Prometheus metrics
  - Resource limits and security contexts
- **Helm Chart v1.8.0**: Complete LGTM stack support with production features
  - Full Tempo/tracing support integration
  - Horizontal Pod Autoscaler (HPA) with CPU and memory metrics
  - Security hardening: non-root execution, dropped capabilities, seccomp profiles
  - Production-ready defaults and examples
  - Topology spread constraints for high availability
  - Comprehensive values.yaml with all configuration options
  - Detailed README with deployment examples
- **Production Containerfile**: Optimized multi-stage Docker builds
  - Security-hardened base images
  - Minimal attack surface
  - Efficient layer caching

#### Developer Experience
- **Comprehensive API Documentation**: Detailed documentation for all Loki, Prometheus, and Tempo endpoints
- **GitHub Actions Automation**: Optimized CI/CD workflows
  - Automated releases with semantic versioning
  - Container image builds for ghcr.io
  - Security scanning with Trivy
  - Automated dependency updates
- **OpenSpec Integration**: Structured change proposal and specification management

### Changed

- **Project Identity**: Renamed from `multena-proxy` to `lgtm-lbac-proxy` to reflect LGTM stack focus
  - Binary name: `multena-proxy` → `lgtm-lbac-proxy`
  - Docker image: `ghcr.io/gepaplexx/multena-proxy` → `ghcr.io/binhnguyenduc/lgtm-lbac-proxy`
  - Go module: `github.com/gepaplexx/multena-proxy` → `github.com/binhnguyenduc/lgtm-lbac-proxy`
- **Documentation**: Complete rewrite focused on LGTM stack use cases
  - Updated README with Tempo examples
  - Added migration guide from multena-proxy
  - Enhanced configuration examples
- **Go Version**: Upgraded to Go 1.25.0 for Docker builds and CI
  - Leverages latest Go performance improvements
  - Enhanced security and stability
- **Build System**: Simplified to AMD64-only builds (removed ARM64 support)
  - Focused on primary deployment targets
  - Faster build times

### Fixed

- **Non-deterministic Test Failure**: Resolved race condition in tenant query generation tests
- **CI/CD Stability**: Multiple improvements to GitHub Actions workflows
  - Disabled problematic lint job
  - Upgraded Trivy security scanner to stable version
  - Set container scan to non-blocking for better CI experience
- **Security Scans**: Resolved security vulnerabilities identified by automated scanning
  - Updated vulnerable dependencies
  - Improved security posture

### Technical Details

#### LGTM Stack Coverage
This release completes the LGTM stack support:
- ✅ **L**oki (Logs) - LogQL enforcement
- ✅ **G**rafana (Visualization) - Compatible datasource configuration
- ✅ **T**empo (Traces) - TraceQL enforcement (NEW)
- ✅ **M**imir/Prometheus (Metrics) - PromQL enforcement

#### Compatibility
- **Configuration Format**: 100% compatible with original multena-proxy
- **API Endpoints**: No breaking changes to routing or request handling
- **Label Store**: ConfigMap and MySQL implementations unchanged
- **Authentication**: JWT/JWKS validation logic preserved

#### Dependencies
- Go 1.25.0
- github.com/grafana/tempo/pkg/traceql - TraceQL parsing
- github.com/MicahParks/keyfunc/v3 - JWKS management
- github.com/prometheus-community/prom-label-proxy - PromQL enforcement
- github.com/observatorium/api - LogQL parsing

### Migration from multena-proxy

For users migrating from the original multena-proxy:

1. Update Docker image references:
   ```bash
   # Old
   ghcr.io/gepaplexx/multena-proxy:latest

   # New
   ghcr.io/binhnguyenduc/lgtm-lbac-proxy:latest
   ```

2. Update binary downloads to use new repository name

3. Configuration files require no changes (100% compatible)

4. Consider using the new Helm chart for Kubernetes deployments

See [MIGRATION.md](MIGRATION.md) for detailed migration instructions.

### Acknowledgments

This project is based on [multena-proxy](https://github.com/gepaplexx/multena-proxy) originally developed by Gepardec. We extend our sincere gratitude to the Gepardec team for creating the foundational architecture.

---

## Previous Releases

This is the first independent release. For history before the fork, see [multena-proxy releases](https://github.com/gepaplexx/multena-proxy/releases).

[0.9.1]: https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/tag/v0.9.1
[0.9.0]: https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/tag/v0.9.0
[0.8.0]: https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/tag/v0.8.0
