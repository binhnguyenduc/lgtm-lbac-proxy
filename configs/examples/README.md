# OAuth Provider Configuration Examples

This directory contains configuration examples for common OAuth providers. Each example shows how to configure JWT claim mappings for the specific provider.

## Available Examples

| Provider | File | Key Differences |
|----------|------|----------------|
| **Keycloak** | [keycloak.yaml](keycloak.yaml) | Standard OpenID Connect claims |
| **Azure AD** | [azure-ad.yaml](azure-ad.yaml) | Uses `unique_name`, `upn`, `roles` |
| **Auth0** | [auth0.yaml](auth0.yaml) | Namespaced custom claims |
| **Google** | [google.yaml](google.yaml) | Uses `email` for username, `hd` for domain |

## JWT Claim Mapping

Different OAuth providers use different claim names for user identity information. The `auth.claims` section allows you to map these provider-specific claims to the proxy's expected fields:

```yaml
auth:
  auth_scheme: "Bearer"           # Token prefix (use "" for raw tokens)
  claims:
    username: "preferred_username"  # JWT claim for username
    email: "email"                   # JWT claim for email
    groups: "groups"                 # JWT claim for groups
```

## Common Claim Names by Provider

### Username Claim

| Provider | Common Claims | Notes |
|----------|---------------|-------|
| Keycloak | `preferred_username` | Default OpenID Connect |
| Azure AD | `unique_name`, `upn`, `name` | User Principal Name recommended |
| Auth0 | `nickname`, `name`, `sub` | Often `nickname` for display |
| Google | `email`, `sub` | `sub` is opaque ID, `email` more readable |

### Email Claim

| Provider | Common Claims | Notes |
|----------|---------------|-------|
| Keycloak | `email` | Standard claim |
| Azure AD | `email`, `upn` | UPN may be email format |
| Auth0 | `email` | Standard claim |
| Google | `email` | Standard claim, includes `email_verified` |

### Groups Claim

| Provider | Common Claims | Notes |
|----------|---------------|-------|
| Keycloak | `groups`, `roles` | Configure in realm settings |
| Azure AD | `roles`, `groups` | Requires app manifest configuration |
| Auth0 | Custom namespaced claim | e.g., `https://domain.com/groups` |
| Google | `hd` (hosted domain) | Limited group support, domain-based |

## Migration from Legacy Configuration

**Old Format (Deprecated):**
```yaml
web:
  jwks_cert_url: "https://oauth.example.com/certs"
  oauth_group_name: "groups"
```

**New Format (Recommended):**
```yaml
auth:
  jwks_cert_url: "https://oauth.example.com/certs"
  claims:
    username: "preferred_username"
    email: "email"
    groups: "groups"
```

## Usage

1. Choose the example that matches your OAuth provider
2. Copy the example to `configs/config.yaml`
3. Update the `jwks_cert_url` with your provider's JWKS endpoint
4. Adjust claim names if your provider uses different fields
5. Test with a sample JWT token from your provider

## Testing Your Configuration

To verify your JWT claim configuration:

1. **Enable debug logging:**
   ```yaml
   log:
     level: 0  # Debug level
     log_tokens: true  # Show token claims (development only!)
   ```

2. **Send a test request** with a JWT token from your provider

3. **Check the logs** for JWT claims extraction:
   ```
   INF JWT claims extracted username=alice@example.com email=alice@example.com groups=[team-a, team-b]
   ```

4. **Disable token logging** in production:
   ```yaml
   log:
     level: 1  # Info level
     log_tokens: false
   ```

## Troubleshooting

### Issue: "No username found in token"

**Solution:** Check your `auth.claims.username` matches the actual claim in your JWT token. Use `log_tokens: true` to see available claims.

### Issue: "No groups found in token"

**Solution:**
- Verify your OAuth provider is configured to include groups in tokens
- For Azure AD: Configure group claims in app manifest
- For Auth0: Add groups via Rules or Actions
- Check the claim name matches your `auth.claims.groups` configuration

### Issue: "Invalid token signature"

**Solution:** Verify your `auth.jwks_cert_url` points to the correct JWKS endpoint for your OAuth provider.

## Additional Resources

- [Keycloak Documentation](https://www.keycloak.org/docs/latest/server_admin/#_oidc)
- [Azure AD Token Claims](https://learn.microsoft.com/en-us/azure/active-directory/develop/access-tokens)
- [Auth0 Custom Claims](https://auth0.com/docs/secure/tokens/json-web-tokens/create-custom-claims)
- [Google OAuth Claims](https://developers.google.com/identity/protocols/oauth2/openid-connect#obtainuserinfo)
