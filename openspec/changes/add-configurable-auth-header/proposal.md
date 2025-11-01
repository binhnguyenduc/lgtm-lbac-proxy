# Proposal: Add Configurable Authentication Header

## Summary
Make the authentication header name configurable instead of hardcoded "Authorization", allowing deployments to use custom header names for JWT token authentication.

## Motivation

### Problem Statement
Currently, the proxy hardcodes the "Authorization" header name for JWT token extraction. This creates limitations for:
- Deployments behind proxies that use different header names
- Integration with systems that pass tokens in custom headers
- Environments where "Authorization" header is reserved or modified by intermediaries

### Current Behavior
- Primary authentication always expects "Authorization" header (hardcoded in `auth.go`)
- Fallback to `alert.token_header` only when alert mode is enabled
- No way to configure the primary authentication header name

### Proposed Solution
Add a configurable `auth_header` field to `WebConfig` that defaults to "Authorization" for backward compatibility, allowing deployments to specify custom header names.

## Impact Analysis

### Breaking Changes
None - defaults to "Authorization" maintaining backward compatibility

### Security Considerations
- No security impact as this only changes the header name, not the authentication logic
- JWT validation and token parsing remain unchanged
- Maintains the same security guarantees

### Performance Impact
Negligible - only changes which HTTP header is read

## Success Criteria
- [ ] Authentication header name is configurable via `web.auth_header` config
- [ ] Defaults to "Authorization" when not specified
- [ ] Existing deployments work without configuration changes
- [ ] Alert mode fallback still works with custom primary header
- [ ] Tests pass with both default and custom header names

## References
- Current implementation: `auth.go:27` (hardcoded "Authorization")
- Alert header pattern: `config.go:42` (AlertConfig.TokenHeader)
- Related config structure: `config.go:23-33` (WebConfig)