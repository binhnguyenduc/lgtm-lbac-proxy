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

func TestTraceQLEnforceMulti(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		policy         LabelPolicy
		expectedResult string
		expectErr      bool
		errorContains  string
	}{
		{
			name:  "Empty query with single rule",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" }`,
			expectErr:      false,
		},
		{
			name:  "Empty query with multiple rules AND logic",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" && resource.team="backend" }`,
			expectErr:      false,
		},
		{
			name:  "Empty query with multiple rules OR logic",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "OR",
			},
			expectedResult: `{ resource.namespace="prod" || resource.team="backend" }`,
			expectErr:      false,
		},
		{
			name:  "Empty query with multiple values (regex)",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod", "staging"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace=~"prod|staging" }`,
			expectErr:      false,
		},
		{
			name:  "Simple query with single rule injection",
			query: `{ span.http.status_code = 500 }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" && span.http.status_code = 500 }`,
			expectErr:      false,
		},
		{
			name:  "Simple query with multiple rules injection",
			query: `{ duration > 100ms }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" && resource.team="backend" && duration > 100ms }`,
			expectErr:      false,
		},
		{
			name:  "Query with != operator",
			query: `{ span.http.status_code = 200 }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "!=", Values: []string{"test"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace!="test" && span.http.status_code = 200 }`,
			expectErr:      false,
		},
		{
			name:  "Query with =~ operator (regex match)",
			query: `{ span.http.method = "GET" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=~", Values: []string{"prod|staging"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace=~"prod|staging" && span.http.method = ` + "`GET`" + ` }`,
			expectErr:      false,
		},
		{
			name:  "Query with !~ operator (regex no-match)",
			query: `{ duration > 1s }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "!~", Values: []string{"test|dev"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace!~"test|dev" && duration > 1s }`,
			expectErr:      false,
		},
		{
			name:  "Query with authorized attribute",
			query: `{ resource.namespace = "prod" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod", "staging"}},
				},
				Logic: "AND",
			},
			expectedResult: "{ resource.namespace = `prod` }",
			expectErr:      false,
		},
		{
			name:  "Query with unauthorized attribute",
			query: `{ resource.namespace = "other" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "unauthorized resource.namespace: other",
		},
		{
			name:  "Query with multiple attributes - all authorized",
			query: `{ resource.namespace = "prod" && resource.team = "backend" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod", "staging"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend", "frontend"}},
				},
				Logic: "AND",
			},
			expectedResult: "{ (resource.namespace = `prod`) && (resource.team = `backend`) }",
			expectErr:      false,
		},
		{
			name:  "Query with multiple attributes - one unauthorized",
			query: `{ resource.namespace = "prod" && resource.team = "external" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "unauthorized resource.team: external",
		},
		{
			name:  "Multiple values with special regex characters",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod.us-west-1", "staging.us-east-1"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace=~"prod\.us-west-1|staging\.us-east-1" }`,
			expectErr:      false,
		},
		{
			name:  "Complex query with OR logic policy",
			query: `{ span.http.status_code >= 500 }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"sre"}},
				},
				Logic: "OR",
			},
			expectedResult: `{ resource.namespace="prod" || resource.team="sre" && span.http.status_code >= 500 }`,
			expectErr:      false,
		},
		{
			name:  "Invalid policy - empty rules",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "must have at least one rule",
		},
		{
			name:  "Invalid policy - bad operator",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "==", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "invalid operator",
		},
		{
			name:  "Invalid TraceQL syntax",
			query: `{ invalid syntax`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr: true,
		},
		{
			name:  "No-op query (empty braces)",
			query: "{}",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" }`,
			expectErr:      false,
		},
		{
			name:  "Query with all policy attributes already present",
			query: `{ resource.namespace = "prod" && resource.team = "backend" && span.http.status_code = 500 }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedResult: "{ ((resource.namespace = `prod`) && (resource.team = `backend`)) && (span.http.status_code = 500) }",
			expectErr:      false,
		},
		{
			name:  "Multiple values with != converts to !~ regex",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "!=", Values: []string{"test", "dev"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace!~"test|dev" }`,
			expectErr:      false,
		},
	}

	enforcer := TraceQLEnforcer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := enforcer.EnforceMulti(tt.query, tt.policy)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				normalizedResult := normalizeWhitespace(result)
				normalizedExpected := normalizeWhitespace(tt.expectedResult)
				assert.Equal(t, normalizedExpected, normalizedResult)
			}
		})
	}
}

func TestBuildPolicyQuery(t *testing.T) {
	tests := []struct {
		name           string
		policy         LabelPolicy
		expectedResult string
	}{
		{
			name: "Single rule",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" }`,
		},
		{
			name: "Multiple rules with AND",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedResult: `{ resource.namespace="prod" && resource.team="backend" }`,
		},
		{
			name: "Multiple rules with OR",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "OR",
			},
			expectedResult: `{ resource.namespace="prod" || resource.team="backend" }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPolicyQuery(tt.policy)
			normalizedResult := normalizeWhitespace(result)
			normalizedExpected := normalizeWhitespace(tt.expectedResult)
			assert.Equal(t, normalizedExpected, normalizedResult)
		})
	}
}

func TestBuildRuleFilter(t *testing.T) {
	tests := []struct {
		name           string
		rule           LabelRule
		expectedResult string
	}{
		{
			name: "Single value with = operator",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "=",
				Values:   []string{"prod"},
			},
			expectedResult: `resource.namespace="prod"`,
		},
		{
			name: "Multiple values with = operator (converts to =~)",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "=",
				Values:   []string{"prod", "staging"},
			},
			expectedResult: `resource.namespace=~"prod|staging"`,
		},
		{
			name: "Single value with != operator",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "!=",
				Values:   []string{"test"},
			},
			expectedResult: `resource.namespace!="test"`,
		},
		{
			name: "Multiple values with != operator (converts to !~)",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "!=",
				Values:   []string{"test", "dev"},
			},
			expectedResult: `resource.namespace!~"test|dev"`,
		},
		{
			name: "Single value with =~ operator",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "=~",
				Values:   []string{"prod.*"},
			},
			expectedResult: `resource.namespace=~"prod.*"`,
		},
		{
			name: "Single value with !~ operator",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "!~",
				Values:   []string{"test.*"},
			},
			expectedResult: `resource.namespace!~"test.*"`,
		},
		{
			name: "Multiple values with special regex chars (escaped)",
			rule: LabelRule{
				Name:     "resource.namespace",
				Operator: "=",
				Values:   []string{"prod.us-west-1", "staging.us-east-1"},
			},
			expectedResult: `resource.namespace=~"prod\.us-west-1|staging\.us-east-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRuleFilter(tt.rule)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestValidatePolicyAttributes(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		policy        LabelPolicy
		expectErr     bool
		errorContains string
	}{
		{
			name:  "No attributes in query",
			query: `{ span.http.status_code = 200 }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr: false,
		},
		{
			name:  "Authorized attribute",
			query: `{ resource.namespace = "prod" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod", "staging"}},
				},
				Logic: "AND",
			},
			expectErr: false,
		},
		{
			name:  "Unauthorized attribute",
			query: `{ resource.namespace = "other" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "unauthorized resource.namespace: other",
		},
		{
			name:  "Regex pattern with all authorized values",
			query: `{ resource.namespace =~ "prod|staging" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod", "staging", "dev"}},
				},
				Logic: "AND",
			},
			expectErr: false,
		},
		{
			name:  "Regex pattern with unauthorized value",
			query: `{ resource.namespace =~ "prod|other" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "unauthorized resource.namespace: other",
		},
		{
			name:  "Multiple attributes - all authorized",
			query: `{ resource.namespace = "prod" && resource.team = "backend" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectErr: false,
		},
		{
			name:  "Multiple attributes - one unauthorized",
			query: `{ resource.namespace = "prod" && resource.team = "external" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "unauthorized resource.team: external",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePolicyAttributes(tt.query, tt.policy)
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

func TestCheckPolicyAttributes(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		policy         LabelPolicy
		expectedResult bool
	}{
		{
			name:  "No attributes in query",
			query: `{ span.http.status_code = 200 }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: false,
		},
		{
			name:  "All attributes present",
			query: `{ resource.namespace = "prod" && resource.team = "backend" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedResult: true,
		},
		{
			name:  "Some attributes missing",
			query: `{ resource.namespace = "prod" }`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "resource.namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "resource.team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPolicyAttributes(tt.query, tt.policy)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
