package main

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestLogQLEnforcer_Enforce(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		policy         LabelPolicy
		expectedResult string
		expectErr      bool
	}{
		{
			name:  "Empty query with single rule",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace="prod"}`,
			expectErr:      false,
		},
		{
			name:  "Empty query with multiple rules",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "environment", Operator: "=", Values: []string{"staging"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace="prod", environment="staging"}`,
			expectErr:      false,
		},
		{
			name:  "Empty query with multiple values - regex OR",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod", "staging", "dev"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace=~"prod|staging|dev"}`,
			expectErr:      false,
		},
		{
			name:  "Simple stream selector - inject namespace",
			query: `{job="app"}`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `{job="app", namespace="prod"}`,
			expectErr:      false,
		},
		{
			name:  "Query with existing namespace - validate allowed",
			query: `{job="app", namespace="prod"}`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod", "staging"}},
				},
				Logic: "AND",
			},
			expectedResult: `{job="app", namespace="prod"}`,
			expectErr:      false,
		},
		{
			name:  "Query with unauthorized namespace",
			query: `{job="app", namespace="other"}`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr: true,
		},
		{
			name:  "Multi-label injection",
			query: `{job="app"}`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "team", Operator: "=", Values: []string{"backend"}},
					{Name: "environment", Operator: "=", Values: []string{"production"}},
				},
				Logic: "AND",
			},
			expectedResult: `{job="app", namespace="prod", team="backend", environment="production"}`,
			expectErr:      false,
		},
		{
			name:  "NotEquals operator",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "!=", Values: []string{"system"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace!="system"}`,
			expectErr:      false,
		},
		{
			name:  "Regex match operator",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=~", Values: []string{"prod.*"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace=~"prod.*"}`,
			expectErr:      false,
		},
		{
			name:  "Regex no-match operator",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "!~", Values: []string{"test.*"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace!~"test.*"}`,
			expectErr:      false,
		},
		{
			name:  "Multi-value NotEquals - regex OR",
			query: "",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "!=", Values: []string{"system", "kube-system"}},
				},
				Logic: "AND",
			},
			expectedResult: `{namespace!~"system|kube-system"}`,
			expectErr:      false,
		},
		{
			name:  "Cluster-wide access - skip enforcement",
			query: `{job="app"}`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "#cluster-wide", Operator: "=", Values: []string{"true"}},
				},
				Logic: "AND",
			},
			expectedResult: `{job="app"}`,
			expectErr:      false,
		},
		{
			name:  "Complex query with rate - inject labels",
			query: `rate({job="app"}[5m])`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `rate({job="app", namespace="prod"}[5m])`,
			expectErr:      false,
		},
		{
			name:  "Query with aggregation - inject labels",
			query: `sum by (job) (rate({job="app"}[5m]))`,
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectedResult: `sum by(job) (rate({job="app", namespace="prod"}[5m]))`,
			expectErr:      false,
		},
	}

	enforcer := LogQLEnforcer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := enforcer.Enforce(tt.query, tt.policy)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestEnforceMultiLabelMatchers(t *testing.T) {
	tests := []struct {
		name           string
		matchers       []*labels.Matcher
		policy         LabelPolicy
		expectedCount  int
		expectErr      bool
		errorContains  string
	}{
		{
			name:     "No existing matchers - inject all rules",
			matchers: []*labels.Matcher{},
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedCount: 2,
			expectErr:     false,
		},
		{
			name: "Existing namespace - validate allowed",
			matchers: []*labels.Matcher{
				{Type: labels.MatchEqual, Name: "namespace", Value: "prod"},
			},
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod", "staging"}},
				},
				Logic: "AND",
			},
			expectedCount: 1,
			expectErr:     false,
		},
		{
			name: "Existing namespace - unauthorized",
			matchers: []*labels.Matcher{
				{Type: labels.MatchEqual, Name: "namespace", Value: "other"},
			},
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
				},
				Logic: "AND",
			},
			expectErr:     true,
			errorContains: "unauthorized namespace: other",
		},
		{
			name: "Partial match - inject missing labels",
			matchers: []*labels.Matcher{
				{Type: labels.MatchEqual, Name: "job", Value: "app"},
				{Type: labels.MatchEqual, Name: "namespace", Value: "prod"},
			},
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: "=", Values: []string{"prod"}},
					{Name: "team", Operator: "=", Values: []string{"backend"}},
				},
				Logic: "AND",
			},
			expectedCount: 3, // job + namespace + team
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EnforceMultiLabelMatchers(tt.matchers, tt.policy)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(result))
			}
		})
	}
}

