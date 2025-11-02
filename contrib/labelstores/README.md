# Community Label Stores

This directory contains community-contributed label store implementations for LGTM LBAC Proxy.

## Official Label Store

The proxy includes a single official label store implementation:
- **FileLabelStore**: File-based YAML label storage (see `labelstore.go`)

## Custom Label Stores

You can implement custom label stores for different backends by implementing the `Labelstore` interface.

### Interface Requirements

All label store implementations must implement the `Labelstore` interface:

```go
type Labelstore interface {
    // Connect establishes a connection with the label store using provided configuration.
    Connect(config LabelStoreConfig) error

    // GetLabels retrieves tenant labels for the given user identity.
    GetLabels(identity UserIdentity) (map[string]bool, bool)
}
```

### Supporting Types

```go
// UserIdentity represents the minimal identity information needed for label lookup.
type UserIdentity struct {
    Username string   // Primary user identifier
    Groups   []string // Group memberships for the user
}

// LabelStoreConfig contains configuration needed by label stores during initialization.
type LabelStoreConfig struct {
    ConfigPaths []string // Paths to search for label configuration files
}
```

### Implementation Guidelines

1. **Connect Method**:
   - Initialize connections to your backend (database, LDAP, API, etc.)
   - Read configuration from `LabelStoreConfig` parameter (focused configuration subset)
   - Use `config.ConfigPaths` for file-based stores to search for configuration files
   - Return error if connection fails or configuration is invalid
   - Use structured logging with `log` from `github.com/rs/zerolog/log`

2. **GetLabels Method**:
   - Extract user/group information from `UserIdentity` parameter:
     - `identity.Username`: Primary user identifier
     - `identity.Groups`: List of group memberships
   - Query your backend for allowed tenant labels
   - Return `(nil, true)` if user has cluster-wide access (`#cluster-wide` label)
   - Return `(map[string]bool, false)` for normal tenant label mappings
   - Return `(empty map, false)` if user has no labels configured
   - Keys in the map are tenant label values (e.g., "prod", "staging", "namespace-a")
   - Values are always `true` in the current implementation

3. **Configuration**:
   - `LabelStoreConfig` is part of the standard configuration system (read from `config.yaml`)
   - Users configure it via `labelstore` section in config.yaml with defaults provided
   - Add custom fields to `LabelStoreConfig` struct in `config.go` using `mapstructure` tags
   - Set defaults in `WithConfig()` method using `v.SetDefault()` or post-unmarshal checks
   - Each label store implementation uses only the fields it needs from `LabelStoreConfig`
   - Document configuration fields in your README with example YAML

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
2. Add custom configuration fields to `LabelStoreConfig` struct in `config.go`
3. Set defaults for your fields in `WithConfig()` method
4. Update `labelstore.go` to import your package
5. Modify `WithLabelStore()` to instantiate your handler and call `Connect(a.Cfg.LabelStore)`
6. Update documentation with usage instructions and configuration examples

**Example Integration**:

```go
// config.go - Add custom fields to LabelStoreConfig
type LabelStoreConfig struct {
    ConfigPaths []string `mapstructure:"config_paths"`

    // Custom fields for your label store
    DatabaseURL string `mapstructure:"database_url"`
    CacheTTL    int    `mapstructure:"cache_ttl"`
}

// config.go - Set defaults in WithConfig()
func (a *App) WithConfig() *App {
    v := viper.NewWithOptions(viper.KeyDelimiter("::"))
    // ... existing code ...

    // Set defaults
    v.SetDefault("labelstore::config_paths", []string{"/etc/config/labels/", "./configs"})
    v.SetDefault("labelstore::cache_ttl", 300)

    // ... rest of method ...
}

// labelstore.go - Update WithLabelStore()
func (a *App) WithLabelStore() *App {
    a.LabelStore = &YourCustomLabelStore{}
    err := a.LabelStore.Connect(a.Cfg.LabelStore)
    if err != nil {
        log.Fatal().Err(err).Msg("Error connecting to labelstore")
    }
    return a
}
```

**Example config.yaml**:

```yaml
labelstore:
  config_paths:
    - /etc/config/labels/
    - ./configs
  database_url: postgres://localhost/labels
  cache_ttl: 600
```

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

- Check the official `FileLabelStore` in `labelstore.go` for reference
- Review the `Labelstore` interface documentation
- Open an issue for questions or guidance
