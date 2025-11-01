# Migration Guide: multena-proxy → lgtm-lbac-proxy

This guide helps existing users migrate from `multena-proxy` to the renamed `lgtm-lbac-proxy`.

## What Changed

### Package Names
- **Binary**: `multena-proxy` → `lgtm-lbac-proxy`
- **Go Module**: `github.com/gepaplexx/multena-proxy` → `github.com/binhnguyenduc/lgtm-lbac-proxy`
- **Docker Image**: `ghcr.io/gepaplexx/multena-proxy` → `ghcr.io/binhnguyenduc/lgtm-lbac-proxy`
- **Helm Chart**: `gp-multena` → `lgtm-lbac-proxy`

### What Didn't Change
- ✅ Configuration format (config.yaml)
- ✅ API endpoints and routes
- ✅ Authentication mechanisms
- ✅ Label-based access control logic
- ✅ All features and functionality

## Migration Steps

### 1. Binary Deployment

**Before:**
```bash
./multena-proxy
```

**After:**
```bash
./lgtm-lbac-proxy
```

**Action:** Update any scripts, systemd services, or startup commands that reference the binary name.

### 2. Docker Deployment

**Before:**
```bash
docker pull ghcr.io/gepaplexx/multena-proxy:latest
docker run ghcr.io/gepaplexx/multena-proxy:latest
```

**After:**
```bash
docker pull ghcr.io/binhnguyenduc/lgtm-lbac-proxy:latest
docker run ghcr.io/binhnguyenduc/lgtm-lbac-proxy:latest
```

**Action:** Update your docker-compose.yml or container runtime configurations.

### 3. Kubernetes/Helm Deployment

**Before:**
```yaml
helm install multena-proxy ./helm/gp-multena \
  --set image.repository=ghcr.io/gepaplexx/multena-proxy
```

**After:**
```yaml
helm install lgtm-lbac-proxy ./helm/lgtm-lbac-proxy \
  --set image.repository=ghcr.io/binhnguyenduc/lgtm-lbac-proxy
```

**Action:** Update your values.yaml:

```yaml
# Before
image:
  repository: ghcr.io/gepaplexx/multena-proxy

# After
image:
  repository: ghcr.io/binhnguyenduc/lgtm-lbac-proxy
```

### 4. SystemD Service

If you're using a systemd service file:

**Before:**
```ini
[Service]
ExecStart=/usr/local/bin/multena-proxy
```

**After:**
```ini
[Service]
ExecStart=/usr/local/bin/lgtm-lbac-proxy
```

**Action:**
```bash
sudo systemctl stop multena-proxy
sudo systemctl disable multena-proxy
# Update service file
sudo systemctl daemon-reload
sudo systemctl enable lgtm-lbac-proxy
sudo systemctl start lgtm-lbac-proxy
```

### 5. Monitoring/Alerting

Update any monitoring rules, dashboards, or alerts that reference:
- Binary name in process checks
- Container image names
- Service names

## Configuration

**No changes required** - Your existing `config.yaml` and `labels.yaml` files will work without modification.

## Validation

After migration, verify the proxy is working:

```bash
# Check version
lgtm-lbac-proxy --version

# Check configuration loads
curl http://localhost:8081/healthz

# Test a query
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/v1/query?query=up"
```

## Rollback

If you need to rollback:

1. Stop the new service
2. Restore the old binary/image names
3. Restart with the previous configuration

No data or configuration changes were made, so rollback is straightforward.

## Support

- **Issues**: https://github.com/binhnguyenduc/lgtm-lbac-proxy/issues
- **Documentation**: https://github.com/binhnguyenduc/lgtm-lbac-proxy#readme

## Credits

This project was originally based on [multena-proxy](https://github.com/gepaplexx/multena-proxy) by Gepardec. The rename establishes this as an independent LGTM-focused solution with ongoing development and enhancements.
