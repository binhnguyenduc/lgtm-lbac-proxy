# Implementation Tasks

## Configuration Changes
- [x] Add `AuthHeader string` field to `WebConfig` struct with mapstructure tag
- [x] Set default value to "Authorization" in config initialization
- [x] Update configuration documentation to include new field

## Code Changes
- [x] Modify `getToken` function to use `a.Cfg.Web.AuthHeader` instead of hardcoded "Authorization"
- [x] Update error messages to reference the configured header name dynamically
- [x] Ensure alert mode fallback still works with custom primary header

## Testing
- [x] Update existing tests to use configurable header
- [x] Add test cases for custom header name configuration
- [x] Test backward compatibility with default "Authorization" header
- [x] Test interaction between custom auth header and alert token header
- [x] Verify error messages correctly reference configured header name

## Documentation
- [x] Update README with new configuration option
- [x] Add example configuration showing custom auth header
- [x] Document migration path for existing deployments (none required)