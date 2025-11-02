package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceQLEnforcer(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		tenantLabels   map[string]bool
		labelMatch     string
		expectedResult string
		expectErr      bool
		errorContains  string
	}{
		{
			name:           "Empty query with single tenant",
			query:          "",
			tenantLabels:   map[string]bool{"prod": true},
			labelMatch:     "resource.namespace",
			expectedResult: `{ resource.namespace="prod" }`,
			expectErr:      false,
		},
		{
			name:           "Empty braces with single tenant",
			query:          "{}",
			tenantLabels:   map[string]bool{"prod": true},
			labelMatch:     "resource.namespace",
			expectedResult: `{ resource.namespace="prod" }`,
			expectErr:      false,
		},
		{
			name:         "Empty query with multiple tenants",
			query:        "",
			tenantLabels: map[string]bool{"prod": true, "staging": true, "dev": true},
			labelMatch:   "resource.namespace",
			// Note: map ordering is not guaranteed, so we check containment instead
			expectedResult: `{ resource.namespace=~"`,
			expectErr:      false,
		},
		{
			name:           "Simple query without tenant label - single tenant",
			query:          `{ span.http.status_code = 200 }`,
			tenantLabels:   map[string]bool{"prod": true},
			labelMatch:     "resource.namespace",
			expectedResult: `{ resource.namespace="prod" && span.http.status_code = 200 }`,
			expectErr:      false,
		},
		{
			name:           "Simple query without tenant label - multiple tenants",
			query:          `{ duration > 100ms }`,
			tenantLabels:   map[string]bool{"prod": true, "staging": true},
			labelMatch:     "resource.namespace",
			expectedResult: `{ resource.namespace=~"prod|staging" && duration > 100ms }`,
			expectErr:      false,
		},
		{
			name:         "Query with authorized tenant label",
			query:        `{ resource.namespace = "prod" }`,
			tenantLabels: map[string]bool{"prod": true, "staging": true},
			labelMatch:   "resource.namespace",
			// Parser converts double quotes to backticks
			expectedResult: "{ resource.namespace = `prod` }",
			expectErr:      false,
		},
		{
			name:          "Query with unauthorized tenant label",
			query:         `{ resource.namespace = "other" }`,
			tenantLabels:  map[string]bool{"prod": true},
			labelMatch:    "resource.namespace",
			expectErr:     true,
			errorContains: "unauthorized resource.namespace: other",
		},
		{
			name:          "Query with regex pattern - unauthorized tenant",
			query:         `{ resource.namespace =~ "prod|other" }`,
			tenantLabels:  map[string]bool{"prod": true},
			labelMatch:    "resource.namespace",
			expectErr:     true,
			errorContains: "unauthorized resource.namespace: other",
		},
		{
			name:         "Query with regex pattern - all authorized tenants",
			query:        `{ resource.namespace =~ "prod|staging" }`,
			tenantLabels: map[string]bool{"prod": true, "staging": true, "dev": true},
			labelMatch:   "resource.namespace",
			// Parser converts to backticks
			expectedResult: "{ resource.namespace =~ `prod|staging` }",
			expectErr:      false,
		},
		{
			name:         "Complex query without tenant label",
			query:        `{ span.http.status_code >= 500 && duration > 1s }`,
			tenantLabels: map[string]bool{"prod": true},
			labelMatch:   "resource.namespace",
			// Parser adds parentheses
			expectedResult: `{ resource.namespace="prod" && (span.http.status_code >= 500) && (duration > 1s) }`,
			expectErr:      false,
		},
		{
			name:         "Query with intrinsic attributes",
			query:        `{ name = "GET /api" }`,
			tenantLabels: map[string]bool{"staging": true},
			labelMatch:   "resource.namespace",
			// Parser converts to backticks
			expectedResult: "{ resource.namespace=\"staging\" && name = `GET /api` }",
			expectErr:      false,
		},
		{
			name:         "Query with multiple conditions and authorized tenant",
			query:        `{ resource.namespace = "prod" && span.http.status_code = 500 }`,
			tenantLabels: map[string]bool{"prod": true},
			labelMatch:   "resource.namespace",
			// Parser adds parentheses and converts to backticks
			expectedResult: "{ (resource.namespace = `prod`) && (span.http.status_code = 500) }",
			expectErr:      false,
		},
		{
			name:         "Invalid TraceQL syntax",
			query:        `{ invalid syntax`,
			tenantLabels: map[string]bool{"prod": true},
			labelMatch:   "resource.namespace",
			expectErr:    true,
		},
		{
			name:         "Query with OR conditions",
			query:        `{ span.http.status_code = 500 || span.http.status_code = 503 }`,
			tenantLabels: map[string]bool{"prod": true},
			labelMatch:   "resource.namespace",
			// Parser adds parentheses
			expectedResult: `{ resource.namespace="prod" && (span.http.status_code = 500) || (span.http.status_code = 503) }`,
			expectErr:      false,
		},
		{
			name:           "Query with backtick-quoted values",
			query:          "{ resource.namespace = `prod` }",
			tenantLabels:   map[string]bool{"prod": true},
			labelMatch:     "resource.namespace",
			expectedResult: "{ resource.namespace = `prod` }",
			expectErr:      false,
		},
		{
			name:         "Tenant label with special regex characters (single tenant, no escaping)",
			query:        "",
			tenantLabels: map[string]bool{"prod.us-west-1": true},
			labelMatch:   "resource.namespace",
			// Single tenant uses equality, no regex escaping needed
			expectedResult: `{ resource.namespace="prod.us-west-1" }`,
			expectErr:      false,
		},
	}

	enforcer := TraceQLEnforcer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := enforcer.Enforce(tt.query, tt.tenantLabels, tt.labelMatch)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// For multi-tenant cases, map ordering is not guaranteed
				// so we just check if result contains expected prefix
				if len(tt.tenantLabels) > 1 && tt.expectedResult == `{ resource.namespace=~"` {
					assert.Contains(t, result, "resource.namespace=~")
					// Verify all tenant values are present
					for tenant := range tt.tenantLabels {
						assert.Contains(t, result, tenant)
					}
				} else {
					// Normalize whitespace for comparison
					normalizedResult := normalizeWhitespace(result)
					normalizedExpected := normalizeWhitespace(tt.expectedResult)
					assert.Equal(t, normalizedExpected, normalizedResult)
				}
			}
		})
	}
}

func TestBuildTenantQuery(t *testing.T) {
	tests := []struct {
		name           string
		labelMatch     string
		tenantLabels   map[string]bool
		expectedResult string
	}{
		{
			name:           "Single tenant",
			labelMatch:     "resource.namespace",
			tenantLabels:   map[string]bool{"prod": true},
			expectedResult: `{ resource.namespace="prod" }`,
		},
		{
			name:           "Multiple tenants",
			labelMatch:     "resource.namespace",
			tenantLabels:   map[string]bool{"prod": true, "staging": true},
			expectedResult: `{ resource.namespace=~"prod|staging" }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTenantQuery(tt.labelMatch, tt.tenantLabels)
			normalizedResult := normalizeWhitespace(result)
			normalizedExpected := normalizeWhitespace(tt.expectedResult)
			assert.Equal(t, normalizedExpected, normalizedResult)
		})
	}
}

func TestBuildTenantFilter(t *testing.T) {
	tests := []struct {
		name         string
		labelMatch   string
		tenantLabels map[string]bool
		expectedOp   string // "=" or "=~"
		checkValues  []string
	}{
		{
			name:         "Single tenant",
			labelMatch:   "resource.namespace",
			tenantLabels: map[string]bool{"prod": true},
			expectedOp:   "=",
			checkValues:  []string{"prod"},
		},
		{
			name:         "Multiple tenants",
			labelMatch:   "resource.namespace",
			tenantLabels: map[string]bool{"prod": true, "staging": true},
			expectedOp:   "=~",
			checkValues:  []string{"prod", "staging"},
		},
		{
			name:         "Tenant with special regex characters (single tenant, no escaping)",
			labelMatch:   "resource.namespace",
			tenantLabels: map[string]bool{"prod.us-west-1": true},
			expectedOp:   "=",
			checkValues:  []string{"prod.us-west-1"}, // No escaping for single tenant
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTenantFilter(tt.labelMatch, tt.tenantLabels)
			assert.Contains(t, result, tt.labelMatch)
			assert.Contains(t, result, tt.expectedOp)
			for _, val := range tt.checkValues {
				assert.Contains(t, result, val)
			}
		})
	}
}

func TestValidateTenantLabels(t *testing.T) {
	tests := []struct {
		name                string
		query               string
		labelMatch          string
		allowedTenantLabels map[string]bool
		expectHasLabel      bool
		expectErr           bool
		errorContains       string
	}{
		{
			name:                "No tenant label in query",
			query:               `{ span.http.status_code = 200 }`,
			labelMatch:          "resource.namespace",
			allowedTenantLabels: map[string]bool{"prod": true},
			expectHasLabel:      false,
			expectErr:           false,
		},
		{
			name:                "Authorized tenant label",
			query:               `{ resource.namespace = "prod" }`,
			labelMatch:          "resource.namespace",
			allowedTenantLabels: map[string]bool{"prod": true},
			expectHasLabel:      true,
			expectErr:           false,
		},
		{
			name:                "Unauthorized tenant label",
			query:               `{ resource.namespace = "other" }`,
			labelMatch:          "resource.namespace",
			allowedTenantLabels: map[string]bool{"prod": true},
			expectHasLabel:      true,
			expectErr:           true,
			errorContains:       "unauthorized resource.namespace: other",
		},
		{
			name:                "Regex pattern with all authorized values",
			query:               `{ resource.namespace =~ "prod|staging" }`,
			labelMatch:          "resource.namespace",
			allowedTenantLabels: map[string]bool{"prod": true, "staging": true},
			expectHasLabel:      true,
			expectErr:           false,
		},
		{
			name:                "Regex pattern with unauthorized value",
			query:               `{ resource.namespace =~ "prod|other" }`,
			labelMatch:          "resource.namespace",
			allowedTenantLabels: map[string]bool{"prod": true},
			expectHasLabel:      true,
			expectErr:           true,
			errorContains:       "unauthorized resource.namespace: other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasLabel, err := validateTenantLabels(tt.query, tt.labelMatch, tt.allowedTenantLabels)
			assert.Equal(t, tt.expectHasLabel, hasLabel)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInjectFilter(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		filter         string
		expectedResult string
	}{
		{
			name:           "Inject into { true }",
			query:          "{ true }",
			filter:         `resource.namespace="prod"`,
			expectedResult: `{ resource.namespace="prod" }`,
		},
		{
			name:           "Inject into simple query",
			query:          `{ span.http.status_code = 200 }`,
			filter:         `resource.namespace="prod"`,
			expectedResult: `{ resource.namespace="prod" && span.http.status_code = 200 }`,
		},
		{
			name:           "Inject into complex query",
			query:          `{ duration > 100ms && span.http.status_code >= 500 }`,
			filter:         `resource.namespace=~"prod|staging"`,
			expectedResult: `{ resource.namespace=~"prod|staging" && duration > 100ms && span.http.status_code >= 500 }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectFilter(tt.query, tt.filter)
			normalizedResult := normalizeWhitespace(result)
			normalizedExpected := normalizeWhitespace(tt.expectedResult)
			assert.Equal(t, normalizedExpected, normalizedResult)
		})
	}
}

func TestEscapeRegexChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No special characters",
			input:    "prod",
			expected: "prod",
		},
		{
			name:     "Dot character",
			input:    "prod.us-west-1",
			expected: `prod\.us-west-1`,
		},
		{
			name:     "Multiple special characters",
			input:    "prod.*+?[]",
			expected: `prod\.\*\+\?\[\]`,
		},
		{
			name:     "Pipe character",
			input:    "prod|staging",
			expected: `prod\|staging`,
		},
		{
			name:     "All special regex characters",
			input:    ".*+?[]()^$|\\",
			expected: `\.\*\+\?\[\]\(\)\^\$\|\\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeRegexChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// normalizeWhitespace removes extra whitespace for comparison
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
