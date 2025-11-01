# Multena Proxy Helm Chart

This directory contains the Helm chart for deploying Multena Proxy to Kubernetes/OpenShift.

## Chart Information

- **Chart Name**: lgtm-lbac-proxy
- **Chart Version**: 1.7.0
- **App Version**: 0.7.0
- **Source**: https://github.com/binhnguyenduc/lgtm-lbac-proxy

## Contents

- `lgtm-lbac-proxy-1.7.0.tgz` - Packaged Helm chart
- `lgtm-lbac-proxy/` - Extracted chart source files

## Installation

### Using the packaged chart

```bash
helm install lgtm-lbac-proxy ./helm/lgtm-lbac-proxy-1.7.0.tgz
```

### Using the source files

```bash
helm install lgtm-lbac-proxy ./helm/lgtm-lbac-proxy
```

## Configuration

Key configuration options in `values.yaml`:

### Proxy Configuration

```yaml
proxy:
  admin:
    bypass: false          # Enable admin bypass
    group: admin-group     # Admin group name

  thanos:
    url: https://thanos-querier:9091
    tenantLabel: namespace

  loki:
    url: https://loki-query-frontend:3100
    tenantLabel: kubernetes_namespace_name
    actorHeader: X-Loki-Actor-Path

  web:
    labelStoreKind: configmap  # configmap or mysql
    oauthGroupName: "groups"
    jwksCertUrl: https://your-oauth-provider/certs
```

### Database Label Store (Optional)

```yaml
proxy:
  db:
    enabled: true
    user: multitenant
    host: localhost
    port: 3306
    dbName: example
    query: "SELECT * FROM users WHERE username = ?"
```

## Grafana Datasources

The chart can automatically create Grafana datasources when using Grafana Operator:

```yaml
GrafanaOperatorDatasources:
  thanos: true
  loki: true
  labelSelector:
    monitoring.gepardec.com/system: 'true'
```

## Notes

- **Tempo Support**: The current chart version (1.7.0) does not include Tempo configuration. To add Tempo support, update the values.yaml with:

```yaml
proxy:
  tempo:
    url: https://tempo-query-frontend:3100
    tenantLabel: resource.namespace
    actorHeader: X-Tempo-User
```

- **OpenShift**: Set `openshift: true` for OpenShift-specific configurations
- **RBAC Collector**: Includes optional RBAC collector for dynamic label management

## Upgrading

To upgrade an existing deployment:

```bash
helm upgrade lgtm-lbac-proxy ./helm/lgtm-lbac-proxy
```

## More Information

- [Main Repository](https://github.com/binhnguyenduc/lgtm-lbac-proxy)
- [Helm Charts Repository](https://github.com/binhnguyenduc/lgtm-lbac-proxy)
- [Documentation](../README.md)
