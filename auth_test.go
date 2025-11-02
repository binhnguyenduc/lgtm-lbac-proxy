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
