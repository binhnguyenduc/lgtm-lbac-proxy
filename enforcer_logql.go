package main

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	logqlv2 "github.com/observatorium/api/logql/v2"
	"github.com/prometheus/prometheus/model/labels"
)

// LogQLEnforcer manipulates and enforces tenant isolation on LogQL queries.
type LogQLEnforcer struct{}

// Enforce modifies a LogQL query string to enforce tenant isolation based on provided tenant labels and a label match string.
// If the input query is empty, a new query is constructed to match provided tenant labels.
// If the input query is non-empty, it is parsed and modified to ensure tenant isolation.
// Returns the modified query or an error if parsing or modification fails.
func (LogQLEnforcer) Enforce(query string, tenantLabels map[string]bool, labelMatch string) (string, error) {
	log.Trace().Str("function", "enforcer").Str("query", query).Msg("input")
	if query == "" {
		operator := "="
		if len(tenantLabels) > 1 {
			operator = "=~"
		}
		query = fmt.Sprintf("{%s%s\"%s\"}", labelMatch, operator, strings.Join(MapKeysToArray(tenantLabels), "|"))
		log.Trace().Str("function", "enforcer").Str("query", query).Msg("enforcing")
		return query, nil
	}
	log.Trace().Str("function", "enforcer").Str("query", query).Msg("enforcing")

	expr, err := logqlv2.ParseExpr(query)
	if err != nil {
		return "", err
	}

	errMsg := error(nil)

	expr.Walk(func(expr interface{}) {
		switch labelExpression := expr.(type) {
		case *logqlv2.StreamMatcherExpr:
			matchers, err := MatchTenantLabelMatchers(labelExpression.Matchers(), tenantLabels, labelMatch)
			if err != nil {
				errMsg = err
				return
			}
			labelExpression.SetMatchers(matchers)
		default:
			// Do nothing
		}
	})
	if errMsg != nil {
		return "", errMsg
	}
	log.Trace().Str("function", "enforcer").Str("query", expr.String()).Msg("enforcing")
	return expr.String(), nil
}

// EnforceMulti modifies a LogQL query string to enforce multi-label policy.
// Supports multiple label rules with different operators (=, !=, =~, !~).
// Handles AND logic by injecting all rules as separate matchers.
// Returns the modified query or an error if parsing/validation fails.
func (LogQLEnforcer) EnforceMulti(query string, policy LabelPolicy) (string, error) {
	log.Trace().Str("function", "enforceMulti").Str("query", query).Msg("input")

	// Validate policy
	if err := policy.Validate(); err != nil {
		return "", fmt.Errorf("invalid policy: %w", err)
	}

	// Check for cluster-wide access
	if policy.HasClusterWideAccess() {
		return query, nil
	}

	// Handle empty query - build from scratch
	if query == "" {
		return buildLogQLQueryFromPolicy(policy), nil
	}

	// Parse existing query
	expr, err := logqlv2.ParseExpr(query)
	if err != nil {
		return "", err
	}

	errMsg := error(nil)

	// Walk AST and inject matchers
	expr.Walk(func(expr interface{}) {
		switch labelExpression := expr.(type) {
		case *logqlv2.StreamMatcherExpr:
			matchers, err := EnforceMultiLabelMatchers(labelExpression.Matchers(), policy)
			if err != nil {
				errMsg = err
				return
			}
			labelExpression.SetMatchers(matchers)
		default:
			// Do nothing
		}
	})

	if errMsg != nil {
		return "", errMsg
	}

	log.Trace().Str("function", "enforceMulti").Str("query", expr.String()).Msg("enforced")
	return expr.String(), nil
}

// buildLogQLQueryFromPolicy constructs a minimal LogQL query from LabelPolicy.
// Combines multiple values for same label using regex OR.
func buildLogQLQueryFromPolicy(policy LabelPolicy) string {
	var matchers []string

	for _, rule := range policy.Rules {
		operator := rule.Operator
		value := rule.Values[0]

		// Combine multiple values with regex OR
		if len(rule.Values) > 1 {
			value = strings.Join(rule.Values, "|")
			if operator == OperatorEquals {
				operator = OperatorRegexMatch
			} else if operator == OperatorNotEquals {
				operator = OperatorRegexNoMatch
			}
		}

		matchers = append(matchers, fmt.Sprintf("%s%s\"%s\"", rule.Name, operator, value))
	}

	return fmt.Sprintf("{%s}", strings.Join(matchers, ", "))
}

// EnforceMultiLabelMatchers enforces multi-label policy on existing matchers.
// Validates existing matchers against policy rules and injects missing ones.
// Returns error if query contains unauthorized label values.
func EnforceMultiLabelMatchers(queryMatches []*labels.Matcher, policy LabelPolicy) ([]*labels.Matcher, error) {
	// Track which rules have been found in the query
	foundRules := make(map[string]bool)

	// Validate existing matchers against policy
	for _, queryMatcher := range queryMatches {
		for _, rule := range policy.Rules {
			if queryMatcher.Name == rule.Name {
				foundRules[rule.Name] = true

				// Validate the matcher's values against the rule
				if err := validateMatcherAgainstRule(queryMatcher, rule); err != nil {
					return nil, err
				}
			}
		}
	}

	// Inject missing rules
	for _, rule := range policy.Rules {
		if !foundRules[rule.Name] {
			matcher := ruleToMatcher(rule)
			queryMatches = append(queryMatches, matcher)
		}
	}

	return queryMatches, nil
}

// validateMatcherAgainstRule checks if a matcher's values are allowed by the rule.
func validateMatcherAgainstRule(matcher *labels.Matcher, rule LabelRule) error {
	// Extract values from matcher (handle regex patterns with |)
	matcherValues := strings.Split(matcher.Value, "|")

	// Check if all matcher values are allowed by the rule
	for _, matcherValue := range matcherValues {
		allowed := false
		for _, ruleValue := range rule.Values {
			if matcherValue == ruleValue {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("unauthorized %s: %s", rule.Name, matcherValue)
		}
	}

	return nil
}

// MatchTenantLabelMatchers ensures tenant label matchers in a LogQL query adhere to provided tenant labels.
// It verifies that the tenant label exists in the query matchers, validating or modifying its values based on tenantLabels.
// If the tenant label is absent in the matchers, it's added along with all values from tenantLabels.
// Returns an error for an unauthorized namespace and nil on success.
func MatchTenantLabelMatchers(queryMatches []*labels.Matcher, tenantLabels map[string]bool, labelMatch string) ([]*labels.Matcher, error) {
	foundTenantLabel := false
	for _, match := range queryMatches {
		if match.Name == labelMatch {
			foundTenantLabel = true
			queryLabels := strings.Split(match.Value, "|")
			for _, queryLabel := range queryLabels {
				_, ok := tenantLabels[queryLabel]
				if !ok {
					return nil, fmt.Errorf("unauthorized label %s", queryLabel)
				}
			}
		}
	}
	if !foundTenantLabel {
		matchType := labels.MatchEqual
		if len(tenantLabels) > 1 {
			matchType = labels.MatchRegexp
		}

		queryMatches = append(queryMatches, &labels.Matcher{
			Type:  matchType,
			Name:  labelMatch,
			Value: strings.Join(MapKeysToArray(tenantLabels), "|"),
		})
	}
	return queryMatches, nil
}
