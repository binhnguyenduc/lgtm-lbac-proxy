# Container Build Guide

This document explains how to build and use the optimized container images for Multena Proxy.

## 📦 Available Containerfiles

### 1. `Containerfile.Build` (Recommended)
Full build from source with multi-stage optimization.

**Use when:**
- Building from scratch
- CI/CD pipelines
- Development builds
- You want the smallest possible image

**Features:**
- Multi-stage build for minimal final image (~10-15 MB)
- Go 1.25 build environment
- Optimized build flags (`-ldflags="-w -s"`, `-trimpath`)
- Layer caching for faster rebuilds
- Security: runs as non-root user (uid:gid = 65532:65532)
- Health checks included
- Version embedding from git tags

### 2. `Containerfile` (Runtime Only)
Packages a pre-built binary.

**Use when:**
- You already have a compiled binary
- Cross-platform builds (build on host, package in container)
- Custom build processes

**Features:**
- Minimal runtime image based on `scratch`
- Requires pre-built `multena-proxy` binary
- Same security and health check features

## 🚀 Quick Start

### Build Everything from Source

```bash
# Build using Containerfile.Build
docker build -f Containerfile.Build -t multena-proxy:latest .

# Or with podman
podman build -f Containerfile.Build -t multena-proxy:latest .
```

### Build with Pre-compiled Binary

```bash
# Step 1: Build the binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -trimpath \
  -o multena-proxy .

# Step 2: Build container
docker build -f Containerfile -t multena-proxy:latest .
```

## 🏗️ Production Build Best Practices

### Multi-Architecture Builds

```bash
# Build for multiple platforms
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Containerfile.Build \
  -t multena-proxy:latest \
  --push \
  .
```

### Build with Version Tags

```bash
# Automatically embed version from git
docker build \
  -f Containerfile.Build \
  -t multena-proxy:$(git describe --tags --always) \
  -t multena-proxy:latest \
  .
```

### Build Arguments (Future Enhancement)

```bash
# Example for future build args support
docker build \
  -f Containerfile.Build \
  --build-arg VERSION=$(git describe --tags) \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VCS_REF=$(git rev-parse --short HEAD) \
  -t multena-proxy:latest \
  .
```

## 🔍 Image Inspection

### View Image Layers

```bash
docker history multena-proxy:latest
```

### Inspect Metadata

```bash
docker inspect multena-proxy:latest
```

### Check Image Size

```bash
docker images multena-proxy:latest
```

Expected size: **~10-15 MB** (Containerfile.Build)

## 🧪 Testing the Image

### Run Container

```bash
docker run --rm \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/configs:/etc/config/config:ro \
  multena-proxy:latest
```

### Health Check

```bash
# Container will automatically health check every 30s
# Manual check:
docker exec <container-id> /usr/local/bin/multena-proxy --version
```

### Run with Custom Config

```bash
docker run --rm \
  -p 8080:8080 \
  -v $(pwd)/configs/config.yaml:/etc/config/config/config.yaml:ro \
  -v $(pwd)/configs/labels.yaml:/etc/config/config/labels.yaml:ro \
  multena-proxy:latest
```

## 🛡️ Security Features

- **Non-root user**: Runs as uid:gid 65532:65532
- **Minimal attack surface**: Based on `scratch` (no shell, no package manager)
- **Static binary**: No external dependencies
- **Read-only filesystem**: Designed to run with read-only root filesystem
- **No secrets in image**: All config via volume mounts

### Security Scanning

```bash
# Scan with Trivy
trivy image multena-proxy:latest

# Scan with Grype
grype multena-proxy:latest

# Scan with Docker Scout
docker scout cves multena-proxy:latest
```

## 📊 Build Optimization Tips

### Layer Caching

The Containerfiles are optimized for Docker layer caching:

1. **Dependencies first**: `go.mod` and `go.sum` copied before source code
2. **Separate stages**: Build and runtime stages are independent
3. **Minimal changes**: Only changed layers are rebuilt

### Build Context Optimization

The `.dockerignore` file excludes:
- Git history and metadata
- Documentation files
- Test files and coverage reports
- IDE configuration
- Temporary and build artifacts
- AI-related folders

Result: **Faster builds** and **smaller build context**

### Clean Builds

```bash
# Remove all build cache
docker builder prune -a

# Force rebuild without cache
docker build --no-cache -f Containerfile.Build -t multena-proxy:latest .
```

## 🐞 Troubleshooting

### Build Fails at Go Modules Download

**Problem**: Network issues or proxy configuration

**Solution**:
```bash
# Use build args for proxy
docker build \
  --build-arg HTTP_PROXY=http://proxy:8080 \
  --build-arg HTTPS_PROXY=http://proxy:8080 \
  -f Containerfile.Build \
  -t multena-proxy:latest \
  .
```

### Binary Not Executable

**Problem**: Permission issues with pre-built binary

**Solution**:
```bash
chmod +x multena-proxy
```

### Health Check Failing

**Problem**: Binary doesn't support `--version` flag

**Solution**: Update Containerfile health check to use actual endpoint:
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s \
  CMD ["/usr/local/bin/multena-proxy", "healthz"]
```

## 📈 Performance Benchmarks

Expected build times (on modern hardware):

- **First build**: 2-5 minutes (depending on network)
- **Cached rebuild** (no code changes): <10 seconds
- **Cached rebuild** (code changes only): 30-60 seconds

Image sizes:
- **Containerfile.Build**: ~10-15 MB
- **Containerfile**: ~8-12 MB

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
- name: Build Container
  run: |
    docker build \
      -f Containerfile.Build \
      -t ghcr.io/${{ github.repository }}:${{ github.sha }} \
      -t ghcr.io/${{ github.repository }}:latest \
      .
```

### GitLab CI Example

```yaml
build:
  image: docker:latest
  script:
    - docker build -f Containerfile.Build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
```

## 📚 Additional Resources

- [Docker Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Docker BuildKit](https://docs.docker.com/build/buildkit/)
- [Container Security Best Practices](https://snyk.io/blog/10-docker-image-security-best-practices/)
- [Distroless Images](https://github.com/GoogleContainerTools/distroless)
