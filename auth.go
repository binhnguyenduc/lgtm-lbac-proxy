package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/golang-jwt/jwt/v5"
)

// OAuthToken represents the structure of an OAuth token.
// It holds user-related information extracted from the token.
type OAuthToken struct {
	Groups            []string `json:"-,omitempty"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	jwt.RegisteredClaims
}

// UserIdentity represents the minimal identity information needed for label lookup.
// This is authentication-agnostic and focuses on the authorization concern.
type UserIdentity struct {
	Username string   // Primary user identifier
	Groups   []string // Group memberships for the user
}

// ToIdentity extracts the identity information from an OAuth token.
// This helper encapsulates the conversion from authentication-specific
// token format to the authorization-focused identity representation.
func (t OAuthToken) ToIdentity() UserIdentity {
	return UserIdentity{
		Username: t.PreferredUsername,
		Groups:   t.Groups,
	}
}

// getToken retrieves the OAuth token from the incoming HTTP request.
// It extracts, parses, and validates the token from the configured authentication header.
func getToken(r *http.Request, a *App) (OAuthToken, error) {
	authToken := r.Header.Get(a.Cfg.Web.AuthHeader)
	if authToken == "" {
		if a.Cfg.Alert.Enabled && r.Header.Get(a.Cfg.Alert.TokenHeader) != "" {
			authToken = r.Header.Get(a.Cfg.Alert.TokenHeader)
		} else {
			return OAuthToken{}, fmt.Errorf("no %s header found", a.Cfg.Web.AuthHeader)
		}
	}
	log.Trace().Str("authToken", authToken).Msg("AuthToken")
	splitToken := strings.Split(authToken, "Bearer")
	log.Trace().Strs("splitToken", splitToken).Msg("SplitToken")
	if len(splitToken) != 2 {
		return OAuthToken{}, fmt.Errorf("invalid %s header", a.Cfg.Web.AuthHeader)
	}

	oauthToken, token, err := parseJwtToken(strings.TrimSpace(splitToken[1]), a)
	if err != nil {
		return OAuthToken{}, fmt.Errorf("error parsing token")
	}
	if !token.Valid {
		return OAuthToken{}, fmt.Errorf("invalid token")
	}
	return oauthToken, nil
}

// parseJwtToken parses the JWT token string and constructs an OAuthToken from the parsed claims.
// It returns the constructed OAuthToken, the parsed jwt.Token, and any error that occurred during parsing.
func parseJwtToken(tokenString string, a *App) (OAuthToken, *jwt.Token, error) {
	var oAuthToken OAuthToken
	var claimsMap jwt.MapClaims

	token, err := jwt.ParseWithClaims(tokenString, &claimsMap, a.Jwks.Keyfunc)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing token")
		return oAuthToken, nil, err
	}

	if !token.Valid {
		log.Trace().Msg("Token is invalid")
	}

	if v, ok := claimsMap["preferred_username"].(string); ok {
		oAuthToken.PreferredUsername = v
		log.Trace().Str("preferred_username", v).Msg("PreferredUsername")
	}

	if v, ok := claimsMap["email"].(string); ok {
		if !strings.Contains(v, "@") {
			log.Warn().Str("email", v).Msg("Email does not contain '@', therefore not an email. Could be sus")
		}
		log.Trace().Str("email", v).Msg("Email")
		oAuthToken.Email = v
	}

	if v, ok := claimsMap[a.Cfg.Web.OAuthGroupName].([]interface{}); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				log.Trace().Str("group", s).Msg("Group")
				oAuthToken.Groups = append(oAuthToken.Groups, s)
			}
		}
	}

	return oAuthToken, token, err
}

// validateLabelPolicy retrieves and validates the label policy for the user.
// It checks if the user is an admin and skips label enforcement if true.
// Returns the LabelPolicy, a boolean indicating whether label enforcement should be skipped,
// and any error that occurred during validation.
func validateLabelPolicy(token OAuthToken, defaultLabel string, a *App) (*LabelPolicy, bool, error) {
	if isAdmin(token, a) {
		log.Debug().Str("user", token.PreferredUsername).Bool("Admin", true).Msg("Skipping label enforcement")
		return nil, true, nil
	}

	policy, err := a.LabelStore.GetLabelPolicy(token.ToIdentity(), defaultLabel)
	if err != nil {
		return nil, false, fmt.Errorf("error getting label policy: %w", err)
	}

	// Check for cluster-wide access
	if policy.HasClusterWideAccess() {
		log.Debug().Str("user", token.PreferredUsername).Bool("ClusterWide", true).Msg("Skipping label enforcement")
		return nil, true, nil
	}

	log.Debug().Str("user", token.PreferredUsername).Int("rules", len(policy.Rules)).Str("logic", policy.Logic).Msg("Label policy retrieved")

	if len(policy.Rules) < 1 {
		return nil, false, fmt.Errorf("no label rules found")
	}
	return policy, false, nil
}

func isAdmin(token OAuthToken, a *App) bool {
	return ContainsIgnoreCase(token.Groups, a.Cfg.Admin.Group) && a.Cfg.Admin.Bypass
}
