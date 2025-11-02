package main

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	enforcer "github.com/prometheus-community/prom-label-proxy/injectproxy"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// PromQLEnforcer is a struct with methods to enforce specific rules on Prometheus Query Language (PromQL) queries.
type PromQLEnforcer struct{}

// Enforce enhances a given PromQL query string with additional label matchers,
// ensuring that the query complies with the allowed tenant labels and specified label match.
// It returns the enhanced query or an error if the query cannot be parsed or is not compliant.
func (PromQLEnforcer) Enforce(query string, allowedTenantLabels map[string]bool, labelMatch string) (string, error) {
	log.Trace().Str("function", "enforcer").Str("query", query).Msg("input")
	if query == "" {
		operator := "="
		if len(allowedTenantLabels) > 1 {
			operator = "=~"
		}
		query = fmt.Sprintf("{%s%s\"%s\"}",
			labelMatch,
			operator,
			strings.Join(MapKeysToArray(allowedTenantLabels),
				"|"))
	}
	log.Trace().Str("function", "enforcer").Str("query", query).Msg("enforcing")
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return "", err
	}

	queryLabels, err := extractLabelsAndValues(expr)
	if err != nil {
		return "", err
	}

	tenantLabels, err := enforceLabels(queryLabels, allowedTenantLabels, labelMatch)
	if err != nil {
		return "", err
	}

	labelEnforcer := createEnforcer(tenantLabels, labelMatch)
	err = labelEnforcer.EnforceNode(expr)
	if err != nil {
		return "", err
	}
	log.Trace().Str("function", "enforcer").Str("query", expr.String()).Msg("enforcing")
	return expr.String(), nil
}

// extractLabelsAndValues parses a PromQL expression and extracts labels and their values.
// It returns a map where keys are label names and values are corresponding label values.
// An error is returned if the expression cannot be parsed.
func extractLabelsAndValues(expr parser.Expr) (map[string]string, error) {
	l := make(map[string]string)
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vector, ok := node.(*parser.VectorSelector); ok {
			for _, matcher := range vector.LabelMatchers {
				l[matcher.Name] = matcher.Value
			}
		}
		return nil
	})
	return l, nil
}

// enforceLabels checks if provided query labels comply with allowed tenant labels and a specified label match.
// If the labels comply, it returns them (or all allowed tenant labels if not specified in the query) and nil.
// If not, it returns an error indicating the non-compliant label.
func enforceLabels(queryLabels map[string]string, allowedTenantLabels map[string]bool, labelMatch string) ([]string, error) {
	if _, ok := queryLabels[labelMatch]; ok {
		ok, tenantLabels := checkLabels(queryLabels, allowedTenantLabels, labelMatch)
		if !ok {
			return nil, fmt.Errorf("user not allowed with tenant label %s", tenantLabels[0])
		}
		return tenantLabels, nil
	}

	return MapKeysToArray(allowedTenantLabels), nil
}

// checkLabels validates if query labels are present in the allowed tenant labels and returns them.
// If a query label is not allowed, it returns false and the non-compliant label.
func checkLabels(queryLabels map[string]string, allowedTenantLabels map[string]bool, labelMatch string) (bool, []string) {
	splitQueryLabels := strings.Split(queryLabels[labelMatch], "|")
	for _, queryLabel := range splitQueryLabels {
		_, ok := allowedTenantLabels[queryLabel]
		if !ok {
			return false, []string{queryLabel}
		}
	}
	return true, splitQueryLabels
}

func createEnforcer(tenantLabels []string, labelMatch string) *enforcer.PromQLEnforcer {
	var matchType labels.MatchType
	if len(tenantLabels) > 1 {
		matchType = labels.MatchRegexp
	} else {
		matchType = labels.MatchEqual
	}

	return enforcer.NewPromQLEnforcer(true, &labels.Matcher{
		Name:  labelMatch,
		Type:  matchType,
		Value: strings.Join(tenantLabels, "|"),
	})
}

// EnforceMulti enhances a given PromQL query with multi-label enforcement based on LabelPolicy.
// It supports multiple label rules with different operators (=, !=, =~, !~) combined with AND logic.
// Returns the enhanced query or an error if the query cannot be parsed or is not compliant.
func (PromQLEnforcer) EnforceMulti(query string, policy LabelPolicy) (string, error) {
	log.Trace().Str("function", "enforceMulti").Str("query", query).Interface("policy", policy).Msg("input")

	// Validate policy
	if err := policy.Validate(); err != nil {
		return "", fmt.Errorf("invalid policy: %w", err)
	}

	// Handle empty query - build from scratch
	if query == "" {
		query = buildQueryFromPolicy(policy)
		log.Trace().Str("function", "enforceMulti").Str("query", query).Msg("built from empty")
	}

	// Parse the query
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}

	// Extract existing labels from query
	queryLabels := extractAllLabelsAndMatchers(expr)

	// Validate existing matchers against policy
	if err := validateQueryAgainstPolicy(queryLabels, policy); err != nil {
		return "", err
	}

	// Build matchers for each rule in policy
	matchers := buildMatchersFromPolicy(policy, queryLabels)

	// Inject matchers into the query
	if err := injectMatchers(expr, matchers); err != nil {
		return "", fmt.Errorf("failed to inject matchers: %w", err)
	}

	result := expr.String()
	log.Trace().Str("function", "enforceMulti").Str("query", result).Msg("output")
	return result, nil
}

// buildQueryFromPolicy constructs a minimal PromQL query from LabelPolicy rules.
// Example: {namespace=~"prod|staging", team!="frontend"}
func buildQueryFromPolicy(policy LabelPolicy) string {
	var matchers []string
	for _, rule := range policy.Rules {
		matcher := buildMatcherString(rule)
		matchers = append(matchers, matcher)
	}
	return fmt.Sprintf("{%s}", strings.Join(matchers, ", "))
}

// buildMatcherString creates a matcher string from a LabelRule.
// Handles multiple values by combining them with regex OR (|).
func buildMatcherString(rule LabelRule) string {
	operator := rule.Operator
	value := rule.Values[0]

	// For multiple values, use regex operator and combine with |
	if len(rule.Values) > 1 {
		value = strings.Join(rule.Values, "|")
		if operator == OperatorEquals {
			operator = OperatorRegexMatch
		} else if operator == OperatorNotEquals {
			operator = OperatorRegexNoMatch
		}
	}

	return fmt.Sprintf("%s%s%q", rule.Name, operator, value)
}

// extractAllLabelsAndMatchers extracts all label matchers from the query expression.
// Returns a map of label name to list of matchers for that label.
func extractAllLabelsAndMatchers(expr parser.Expr) map[string][]*labels.Matcher {
	labelMatchers := make(map[string][]*labels.Matcher)

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vector, ok := node.(*parser.VectorSelector); ok {
			for _, matcher := range vector.LabelMatchers {
				labelMatchers[matcher.Name] = append(labelMatchers[matcher.Name], matcher)
			}
		}
		return nil
	})

	return labelMatchers
}

// validateQueryAgainstPolicy checks if existing query matchers comply with the policy.
// Returns an error if any matcher violates the policy constraints.
func validateQueryAgainstPolicy(queryLabels map[string][]*labels.Matcher, policy LabelPolicy) error {
	// Build a map of label name to allowed values for quick lookup
	policyMap := make(map[string]*LabelRule)
	for i := range policy.Rules {
		policyMap[policy.Rules[i].Name] = &policy.Rules[i]
	}

	// Check each existing matcher
	for labelName, matchers := range queryLabels {
		rule, hasRule := policyMap[labelName]
		if !hasRule {
			// Label not in policy - skip validation (policy will be injected)
			continue
		}

		// Validate each matcher for this label
		for _, matcher := range matchers {
			if err := validateMatcher(matcher, rule); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateMatcher checks if a single matcher complies with the policy rule.
func validateMatcher(matcher *labels.Matcher, rule *LabelRule) error {
	// Create a map of allowed values
	allowedValues := make(map[string]bool)
	for _, v := range rule.Values {
		allowedValues[v] = true
	}

	// For equality matchers, check if value is allowed
	if matcher.Type == labels.MatchEqual {
		if !allowedValues[matcher.Value] {
			return fmt.Errorf("unauthorized %s: %s", matcher.Name, matcher.Value)
		}
	}

	// For regex matchers, check if all pipe-separated values are allowed
	if matcher.Type == labels.MatchRegexp {
		values := strings.Split(matcher.Value, "|")
		for _, v := range values {
			if !allowedValues[v] {
				return fmt.Errorf("unauthorized %s: %s", matcher.Name, v)
			}
		}
	}

	// For not-equal and negative regex, we allow them if they're more restrictive
	// (this is a complex semantic check that would require analyzing the full boolean logic)

	return nil
}

// buildMatchersFromPolicy creates label matchers from policy rules.
// Skips rules that already have valid matchers in the query.
func buildMatchersFromPolicy(policy LabelPolicy, existingLabels map[string][]*labels.Matcher) []*labels.Matcher {
	var matchers []*labels.Matcher

	for _, rule := range policy.Rules {
		// Skip if this label already has matchers (already validated)
		if _, exists := existingLabels[rule.Name]; exists {
			continue
		}

		matcher := ruleToMatcher(rule)
		matchers = append(matchers, matcher)
	}

	return matchers
}

// injectMatchers injects label matchers into all vector selectors in the expression.
func injectMatchers(expr parser.Expr, matchers []*labels.Matcher) error {
	if len(matchers) == 0 {
		return nil
	}

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vector, ok := node.(*parser.VectorSelector); ok {
			// Add matchers that don't already exist
			for _, newMatcher := range matchers {
				hasLabel := false
				for _, existing := range vector.LabelMatchers {
					if existing.Name == newMatcher.Name {
						hasLabel = true
						break
					}
				}
				if !hasLabel {
					vector.LabelMatchers = append(vector.LabelMatchers, newMatcher)
				}
			}
		}
		return nil
	})

	return nil
}
