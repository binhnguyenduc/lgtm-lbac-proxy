# Community Label Stores

This directory contains community-contributed label store implementations for LGTM LBAC Proxy.

## Official Label Store

The proxy includes a single official label store implementation:
- **ConfigMapHandler**: File-based YAML label storage (see `labelstore.go`)

## Custom Label Stores

You can implement custom label stores for different backends by implementing the `Labelstore` interface.

### Interface Requirements

All label store implementations must implement the `Labelstore` interface:

```go
type Labelstore interface {
    // Connect establishes a connection with the label store using App configuration.
    Connect(App) error

    // GetLabels retrieves labels associated with the provided OAuth token.
    // Returns a map containing the labels and a boolean indicating whether
    // the label is cluster-wide or not.
    GetLabels(token OAuthToken) (map[string]bool, bool)
}
```

### Implementation Guidelines

1. **Connect Method**:
   - Initialize connections to your backend (database, LDAP, API, etc.)
   - Read configuration from `App.Cfg` structure
   - Return error if connection fails
   - Use structured logging with `log` from `github.com/rs/zerolog/log`

2. **GetLabels Method**:
   - Extract user/group information from `OAuthToken` struct:
     - `token.PreferredUsername`: Primary username
     - `token.Email`: User email
     - `token.Groups`: List of group memberships
   - Query your backend for allowed tenant labels
   - Return `(nil, true)` if user has cluster-wide access (`#cluster-wide` label)
   - Return `(map[string]bool, false)` for normal tenant label mappings
   - Keys in the map are tenant label values (e.g., "prod", "staging", "namespace-a")
   - Values are always `true` in the current implementation

3. **Configuration**:
   - Add custom config structs to `config.go`
   - Use viper's `mapstructure` tags for YAML binding
   - Document configuration fields in your README

4. **Error Handling**:
   - Use `log.Fatal()` only for initialization errors in `Connect()`
   - Return empty map and log warnings for user-not-found scenarios
   - Handle connection failures gracefully in `GetLabels()`

5. **Testing**:
   - Provide comprehensive unit tests
   - Include integration tests if applicable
   - Test cluster-wide access scenarios
   - Test error handling and edge cases

### Example Structure

See `example/` directory for a template implementation.

```
contrib/labelstores/
├── README.md                    # This file
├── example/                     # Template implementation
│   ├── README.md               # Implementation-specific docs
│   ├── labelstore.go           # Implementation code
│   ├── labelstore_test.go      # Unit tests
│   └── config.example.yaml     # Configuration example
└── your-implementation/         # Your custom implementation
    ├── README.md
    ├── labelstore.go
    └── labelstore_test.go
```

### Integration

To use a custom label store:

1. Copy your implementation to the project
2. Update `labelstore.go` to import your package
3. Modify `WithLabelStore()` to instantiate your handler
4. Add configuration support to `config.go`
5. Update documentation with usage instructions

### Submission Guidelines

If you'd like to contribute your label store implementation:

1. Place it in `contrib/labelstores/<name>/`
2. Include comprehensive README with:
   - Purpose and use case
   - Configuration instructions
   - Setup requirements
   - Example configuration
3. Ensure all tests pass
4. Submit a pull request with clear description

### Available Implementations

- **example**: Template implementation (not functional)

### Need Help?

- Check the official `ConfigMapHandler` in `labelstore.go` for reference
- Review the `Labelstore` interface documentation
- Open an issue for questions or guidance
