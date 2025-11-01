# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.8.0]: https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/tag/v0.8.0
