# syntax=docker/dockerfile:1

# This Containerfile expects a pre-built multena-proxy binary
# Build the binary first: CGO_ENABLED=0 go build -ldflags="-w -s" -trimpath -o multena-proxy .

# Helper stage to create user and get certificates
FROM alpine:latest AS prep

# Create nonroot user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

# Update CA certificates
RUN apk add --no-cache ca-certificates tzdata

# Final runtime stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/ca/ca-certificates.crt

# Copy timezone data
COPY --from=prep /usr/share/zoneinfo /usr/share/zoneinfo

# Copy user and group files
COPY --from=prep /etc/passwd /etc/passwd
COPY --from=prep /etc/group /etc/group

# Copy the pre-built binary
COPY multena-proxy /usr/local/bin/multena-proxy

# Run as non-root user
USER 65532:65532

# Add metadata labels
LABEL org.opencontainers.image.title="Multena Proxy" \
      org.opencontainers.image.description="Multi-tenancy authorization proxy for LGTM stack" \
      org.opencontainers.image.vendor="Fork by binhnguyenduc" \
      org.opencontainers.image.source="https://github.com/binhnguyenduc/lgtm-lbac-proxy" \
      org.opencontainers.image.licenses="Apache-2.0"

# Health check using the binary itself
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/usr/local/bin/multena-proxy", "--version"]

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/multena-proxy"]
