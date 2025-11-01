# Helm Chart Changelog

All notable changes to the LGTM LBAC Proxy Helm chart will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.8.0] - 2025-01-01

### Added
- ✨ **Tempo Support**: Full Grafana Tempo integration with TraceQL enforcement
  - Added `proxy.tempo` configuration section
  - Added `tls.tempo` for mTLS support
  - Added Tempo volume mounts and secrets handling
  - Added Tempo configuration to ConfigMap generation
- ✨ **Horizontal Pod Autoscaler**: Added HPA template for auto-scaling
  - Configurable via `autoscaling.enabled`
  - Supports CPU and memory-based scaling
  - Min/max replicas configuration
  - Custom behavior configuration support
- 🔒 **Security Hardening**: Comprehensive pod security improvements
  - Pod security context with non-root user (65534)
  - Container security context with dropped capabilities
  - seccomp profile (RuntimeDefault)
  - `allowPrivilegeEscalation: false`
  - Applied to both proxy and collector containers

### Changed
- ⚡ **Resource Management**: Improved resource requests and limits
  - Proxy: Increased requests (50m CPU, 64Mi memory) and added limits (200m CPU, 128Mi memory)
  - Collector: Added CPU limit (50m)
  - Resources now always applied (not conditional on probe configuration)
- 📈 **Revision History**: Increased `revisionHistoryLimit` from 1 to 3 for easier rollbacks
- 🏷️ **Naming Standardization**: Complete naming consistency
  - Changed all `gp-multena` references to `lgtm-lbac-proxy`
  - Container name: `multena-proxy` → `proxy`
  - Volume names simplified: `multena-config` → `config`, `multena-labels` → `labels`
  - TLS volume names: `multena-loki-tls-secret` → `loki-tls-secret`
- 🌐 **Default URLs**: Updated to vanilla Kubernetes conventions
  - Thanos: `openshift-monitoring` → `monitoring` namespace
  - Loki: `openshift-logging` → `logging` namespace
  - Tempo: Added `tracing` namespace default

### Removed
- 🗑️ **OpenShift Support**: Removed OpenShift-specific features
  - Removed `openshift` flag from values.yaml
  - Removed OpenShift service CA volume mounts
  - Removed OpenShift-specific ConfigMap dependencies
  - Chart now targets vanilla Kubernetes only
- 🗑️ **RBAC Collector**: Removed automatic RBAC collection
  - Removed `collector` configuration section
  - Removed `templates/rbac-collector/` directory
  - Removed collector deployment, service, and RBAC resources
  - Users must manually create labels ConfigMap
- 🗑️ **MySQL Label Store**: Removed MySQL database support
  - Removed `proxy.db` configuration section
  - Removed `templates/external-secrets.yaml`
  - Removed MySQL volume mounts and secrets
  - Removed DB-related helper functions
  - Chart now uses ConfigMap-only for label storage
- 🗑️ **Grafana Operator Datasources**: Removed GrafanaDatasource CRD support
  - Removed `GrafanaOperatorDatasources` configuration
  - Removed `templates/system-datasources.yaml`
  - Removed automatic datasource provisioning
- 🗑️ **Kyverno TLS Copy**: Removed Kyverno-based secret copying
  - Removed `tls.copy` configuration section
  - Removed `templates/tls-copy.yaml`
  - Users must manually create TLS secrets in namespace

### Fixed
- 🐛 **Resource Definition Bug**: Fixed resources block placement
  - Resources were incorrectly inside readiness probe conditional
  - Now properly placed at container level
  - Resources always applied regardless of probe configuration

### Infrastructure
- 📦 **Simplified Chart**: Reduced template count
  - Before: 13 templates (including subdirectories)
  - After: 7 core templates
  - Removed external dependencies (Kyverno, External Secrets Operator, Grafana Operator)
- 🎯 **Focused Scope**: Chart now focused on core proxy functionality
  - No platform-specific features
  - No external operator dependencies
  - Pure Kubernetes compatibility

### Migration Notes

**Breaking Changes:**
- **Label Store**: MySQL label store no longer supported. Migrate to ConfigMap-based labels.
- **RBAC Collector**: Automatic label collection removed. Create labels ConfigMap manually.
- **OpenShift**: OpenShift-specific features removed. Use vanilla Kubernetes configuration.
- **External Operators**: Kyverno, External Secrets, and Grafana Operator support removed.

**Configuration Changes:**
```yaml
# Removed configurations:
openshift: true                    # No longer supported
collector.enabled: true            # Removed
proxy.db.*                         # Removed (MySQL)
GrafanaOperatorDatasources.*       # Removed
tls.copy.*                         # Removed (Kyverno)

# New configurations:
proxy.tempo.*                      # Added (Tempo support)
tls.tempo.*                        # Added (Tempo TLS)
autoscaling.*                      # Added (HPA support)
```

**Volume Name Changes:**
```yaml
# Old → New
multena-config           → config
multena-labels           → labels
multena-loki-tls-secret  → loki-tls-secret
multena-thanos-tls-secret → thanos-tls-secret
# New
tempo-tls-secret         → (new for Tempo)
```

**Migration Steps:**
1. Backup current Helm values: `helm get values lgtm-lbac-proxy > backup.yaml`
2. Create labels ConfigMap manually (no longer auto-created by collector)
3. Remove unsupported configuration keys from values
4. Update volume references if using custom configurations
5. Apply upgrade: `helm upgrade lgtm-lbac-proxy ./helm/lgtm-lbac-proxy -f new-values.yaml`

### Acknowledgments

This release represents a major refactoring focused on:
- **Production readiness**: Better security, resources, and scaling
- **Simplicity**: Removed external dependencies and platform-specific features
- **LGTM completeness**: Added Tempo support for full observability stack coverage
- **Maintainability**: Standardized naming and reduced complexity

---

## [1.7.0 and earlier] - Historical

For changes in version 1.7.0 and earlier, see the [multena-proxy](https://github.com/gepaplexx/multena-proxy) upstream project history.

This chart is based on the original multena-proxy Helm chart (v1.7.0) with significant modifications for LGTM stack focus.

**Note**: Version 1.7.0 was the last release of the original multena-proxy chart. This project continues from that baseline as version 1.8.0 with LGTM-focused enhancements.

[1.8.0]: https://github.com/binhnguyenduc/lgtm-lbac-proxy/releases/tag/v1.8.0
