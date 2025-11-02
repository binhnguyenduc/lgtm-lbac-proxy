package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)


// normalizeWhitespace removes extra whitespace for comparison
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func TestTraceQLEnforcer_Enforce(t *testing.T) {
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
			result, err := enforcer.Enforce(tt.query, tt.policy)
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
