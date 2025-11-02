package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetToken_ValidToken(t *testing.T) {
	app, tokens := setupTestMain()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokens["userTenant"])

	token, err := getToken(req, &app)

	assert.NoError(t, err)
	assert.Equal(t, "user", token.PreferredUsername)
	assert.Equal(t, "test@email.com", token.Email)
}

func TestGetToken_MissingAuthorizationHeader(t *testing.T) {
	app, _ := setupTestMain()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token, err := getToken(req, &app)

	assert.Error(t, err)
	assert.Equal(t, OAuthToken{}, token)
}

func TestGetToken_InvalidAuthorizationFormat(t *testing.T) {
	app, _ := setupTestMain()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "InvalidToken")

	token, err := getToken(req, &app)

	assert.Error(t, err)
	assert.Equal(t, OAuthToken{}, token)
}

func TestParseJwtToken_ValidToken(t *testing.T) {
	app, tokens := setupTestMain()
	tokenString := tokens["groupTenant"]

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	assert.Equal(t, "not-a-user", oauthToken.PreferredUsername)
	assert.Equal(t, "test@email.com", oauthToken.Email)
}

func TestParseJwtToken_InvalidToken(t *testing.T) {
	app, _ := setupTestMain()
	tokenString := "invalidToken"

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.Error(t, err)
	assert.Equal(t, OAuthToken{}, oauthToken)
}

func TestIsAdmin_ValidAdminUser(t *testing.T) {
	app, tokens := setupTestMain()
	tokenString := tokens["adminUserToken"]

	oauthToken, _, _ := parseJwtToken(tokenString, &app)

	app.Cfg.Admin.Group = "admins"
	app.Cfg.Admin.Bypass = true

	isAdmin := isAdmin(oauthToken, &app)

	assert.True(t, isAdmin)
}

func TestIsAdmin_NonAdminUser(t *testing.T) {
	app, tokens := setupTestMain()
	tokenString := tokens["userTenant"]

	oauthToken, _, _ := parseJwtToken(tokenString, &app)

	app.Cfg.Admin.Group = "admins"
	app.Cfg.Admin.Bypass = true

	isAdmin := isAdmin(oauthToken, &app)

	assert.False(t, isAdmin)
}

func TestGetToken_CustomAuthHeader(t *testing.T) {
	app, tokens := setupTestMain()
	app.Cfg.Web.AuthHeader = "X-Custom-Auth"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Auth", "Bearer "+tokens["userTenant"])

	token, err := getToken(req, &app)

	assert.NoError(t, err)
	assert.Equal(t, "user", token.PreferredUsername)
	assert.Equal(t, "test@email.com", token.Email)
}

func TestGetToken_CustomAuthHeader_MissingHeader(t *testing.T) {
	app, _ := setupTestMain()
	app.Cfg.Web.AuthHeader = "X-Custom-Auth"
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token, err := getToken(req, &app)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Custom-Auth")
	assert.Equal(t, OAuthToken{}, token)
}

func TestGetToken_CustomAuthHeader_InvalidFormat(t *testing.T) {
	app, _ := setupTestMain()
	app.Cfg.Web.AuthHeader = "X-Custom-Auth"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Auth", "InvalidToken")

	token, err := getToken(req, &app)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Custom-Auth")
	assert.Equal(t, OAuthToken{}, token)
}

func TestGetToken_DefaultAuthHeader(t *testing.T) {
	app, tokens := setupTestMain()
	// AuthHeader should default to "Authorization"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokens["userTenant"])

	token, err := getToken(req, &app)

	assert.NoError(t, err)
	assert.Equal(t, "user", token.PreferredUsername)
	assert.Equal(t, "Authorization", app.Cfg.Web.AuthHeader)
}

func TestGetToken_AlertModeFallback_CustomAuthHeader(t *testing.T) {
	app, tokens := setupTestMain()
	app.Cfg.Web.AuthHeader = "X-Custom-Auth"
	app.Cfg.Alert.Enabled = true
	app.Cfg.Alert.TokenHeader = "X-Alert-Token"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Don't set X-Custom-Auth, but set alert token
	req.Header.Set("X-Alert-Token", "Bearer "+tokens["userTenant"])

	token, err := getToken(req, &app)

	assert.NoError(t, err)
	assert.Equal(t, "user", token.PreferredUsername)
	assert.Equal(t, "test@email.com", token.Email)
}

func TestParseJwtToken_DefaultClaimNames(t *testing.T) {
	app, tokens := setupTestMain()
	tokenString := tokens["userTenant"]

	// Verify default claim names are set
	assert.Equal(t, "preferred_username", app.Cfg.Web.OAuthUsernameClaim)
	assert.Equal(t, "email", app.Cfg.Web.OAuthEmailClaim)
	assert.Equal(t, "groups", app.Cfg.Web.OAuthGroupName)

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	assert.Equal(t, "user", oauthToken.PreferredUsername)
	assert.Equal(t, "test@email.com", oauthToken.Email)
}

func TestParseJwtToken_CustomUsernameClaim_AzureAD(t *testing.T) {
	// Test with Azure AD style claims
	app, _, pk := setupTestMainWithPrivateKey()
	app.Cfg.Web.OAuthUsernameClaim = "unique_name"

	// Create token with Azure AD claim structure
	claims := map[string]interface{}{
		"unique_name": "azure-user",
		"email":       "azure@example.com",
		"groups":      []interface{}{"team1"},
	}
	tokenString, err := genJWKSWithCustomClaims(claims, pk)
	assert.NoError(t, err)

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	assert.Equal(t, "azure-user", oauthToken.PreferredUsername)
	assert.Equal(t, "azure@example.com", oauthToken.Email)
	assert.Equal(t, []string{"team1"}, oauthToken.Groups)
}

func TestParseJwtToken_CustomEmailClaim_AzureAD(t *testing.T) {
	// Test with Azure AD upn (User Principal Name) for email
	app, _, pk := setupTestMainWithPrivateKey()
	app.Cfg.Web.OAuthEmailClaim = "upn"

	claims := map[string]interface{}{
		"preferred_username": "azure-user",
		"upn":                "user@domain.com",
		"groups":             []interface{}{"team1"},
	}
	tokenString, err := genJWKSWithCustomClaims(claims, pk)
	assert.NoError(t, err)

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	assert.Equal(t, "azure-user", oauthToken.PreferredUsername)
	assert.Equal(t, "user@domain.com", oauthToken.Email)
}

func TestParseJwtToken_CustomGroupsClaim_Auth0(t *testing.T) {
	// Test with Auth0 namespaced claims
	app, _, pk := setupTestMainWithPrivateKey()
	app.Cfg.Web.OAuthGroupName = "https://example.com/groups"

	claims := map[string]interface{}{
		"preferred_username":         "auth0-user",
		"email":                      "auth0@example.com",
		"https://example.com/groups": []interface{}{"admin", "user"},
	}
	tokenString, err := genJWKSWithCustomClaims(claims, pk)
	assert.NoError(t, err)

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	assert.Equal(t, "auth0-user", oauthToken.PreferredUsername)
	assert.Equal(t, "auth0@example.com", oauthToken.Email)
	assert.Equal(t, []string{"admin", "user"}, oauthToken.Groups)
}

func TestParseJwtToken_AllCustomClaims(t *testing.T) {
	// Test with all claims customized
	app, _, pk := setupTestMainWithPrivateKey()
	app.Cfg.Web.OAuthUsernameClaim = "sub"
	app.Cfg.Web.OAuthEmailClaim = "mail"
	app.Cfg.Web.OAuthGroupName = "roles"

	claims := map[string]interface{}{
		"sub":   "user-12345",
		"mail":  "custom@example.com",
		"roles": []interface{}{"developer", "admin"},
	}
	tokenString, err := genJWKSWithCustomClaims(claims, pk)
	assert.NoError(t, err)

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	assert.Equal(t, "user-12345", oauthToken.PreferredUsername)
	assert.Equal(t, "custom@example.com", oauthToken.Email)
	assert.Equal(t, []string{"developer", "admin"}, oauthToken.Groups)
}

func TestParseJwtToken_MissingCustomClaim(t *testing.T) {
	// Test when configured claim is missing from token
	app, _, pk := setupTestMainWithPrivateKey()
	app.Cfg.Web.OAuthUsernameClaim = "non_existent_claim"

	claims := map[string]interface{}{
		"preferred_username": "user",
		"email":              "user@example.com",
		"groups":             []interface{}{"team1"},
	}
	tokenString, err := genJWKSWithCustomClaims(claims, pk)
	assert.NoError(t, err)

	oauthToken, _, err := parseJwtToken(tokenString, &app)

	assert.NoError(t, err)
	// Username should be empty since the custom claim doesn't exist
	assert.Equal(t, "", oauthToken.PreferredUsername)
	assert.Equal(t, "user@example.com", oauthToken.Email)
}
