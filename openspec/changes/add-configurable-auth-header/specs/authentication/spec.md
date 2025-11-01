# Authentication Specification Delta

## ADDED Requirements

### Requirement: Configurable Authentication Header
The system SHALL support configurable authentication header names for JWT token extraction.

#### Scenario: Custom Authentication Header
Given a deployment with `web.auth_header` configured as "X-Auth-Token"
When a request arrives with JWT token in "X-Auth-Token" header
Then the system extracts and validates the token from "X-Auth-Token" header

#### Scenario: Default Behavior
Given a deployment without `web.auth_header` configured
When a request arrives with JWT token in "Authorization" header
Then the system extracts and validates the token from "Authorization" header

#### Scenario: Missing Configured Header
Given a deployment with `web.auth_header` configured as "X-Custom-Auth"
When a request arrives without "X-Custom-Auth" header
Then the system returns error "no X-Custom-Auth header found"

## MODIFIED Requirements

### Requirement: Token Extraction from HTTP Headers
The system SHALL extract JWT tokens from the configured authentication header name instead of hardcoded "Authorization".

#### Scenario: Alert Mode with Custom Auth Header
Given a deployment with:
- `web.auth_header` configured as "X-Auth-Token"
- `alert.enabled` set to true
- `alert.token_header` configured as "X-Alert-Token"
When a request arrives without "X-Auth-Token" but with "X-Alert-Token" header
Then the system falls back to extracting token from "X-Alert-Token" header

#### Scenario: Invalid Token Format with Custom Header
Given a deployment with `web.auth_header` configured as "X-Custom-Auth"
When a request arrives with malformed token in "X-Custom-Auth" header
Then the system returns error "invalid X-Custom-Auth header"