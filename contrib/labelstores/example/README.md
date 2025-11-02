# Example Label Store

This is a template implementation showing how to create a custom label store for LGTM LBAC Proxy.

## Purpose

This example demonstrates:
- Implementing the `Labelstore` interface
- Connecting to an external backend
- Retrieving and mapping user/group labels
- Handling cluster-wide access
- Error handling and logging

## Implementation Overview

The example shows a hypothetical external API label store that:
1. Connects to an external API on initialization
2. Queries the API for user/group label mappings
3. Caches results for performance
4. Handles errors gracefully

## Configuration

Add to your `config.yaml`:

```yaml
example_api:
  base_url: "https://api.example.com"
  api_key_path: "/path/to/api/key"
  timeout: 10s
  cache_ttl: 5m
```

## Usage

This is a template only. To use this as a starting point:

1. Copy this directory: `cp -r contrib/labelstores/example contrib/labelstores/my-store`
2. Rename the handler struct (e.g., `MyStoreHandler`)
3. Implement your backend connection logic
4. Update configuration struct in `config.go`
5. Add tests for your implementation
6. Update this README with your specific documentation

## Testing

```bash
go test ./contrib/labelstores/my-store/...
```

## Reference

See `labelstore.go` in the project root for the official `ConfigMapHandler` implementation.
