package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/rs/zerolog/log"
)

// TraceQLEnforcer manipulates and enforces tenant isolation on TraceQL queries.
type TraceQLEnforcer struct{}

// Enforce modifies a TraceQL query string to enforce tenant isolation based on provided tenant labels and a label match string.
// If the input query is empty, a new query is constructed to match provided tenant labels.
// If the input query is non-empty, it is parsed and modified to ensure tenant isolation.
// Returns the modified query or an error if parsing or modification fails.
func (TraceQLEnforcer) Enforce(query string, tenantLabels map[string]bool, labelMatch string) (string, error) {
	log.Trace().Str("function", "enforcer").Str("query", query).Msg("input")

	// Handle empty query or just braces
	if query == "" || strings.TrimSpace(query) == "{}" {
		query = buildTenantQuery(labelMatch, tenantLabels)
		log.Trace().Str("function", "enforcer").Str("query", query).Msg("enforcing empty query")
		return query, nil
	}

	log.Trace().Str("function", "enforcer").Str("query", query).Msg("enforcing")

	// Parse query to validate syntax
	ast, err := traceql.Parse(query)
	if err != nil {
		return "", fmt.Errorf("invalid TraceQL syntax: %w", err)
	}

	// Check if it's a no-op query (e.g., "{ true }")
	if ast.IsNoop() {
		query = buildTenantQuery(labelMatch, tenantLabels)
		log.Trace().Str("function", "enforcer").Str("query", query).Msg("enforcing noop query")
		return query, nil
	}

	// Get serialized version for manipulation
	serialized := ast.String()

	// Check if query already contains the tenant label and validate it
	hasTenantLabel, err := validateTenantLabels(serialized, labelMatch, tenantLabels)
	if err != nil {
		return "", err
	}

	// If query already has valid tenant label, return it as-is
	if hasTenantLabel {
		log.Trace().Str("function", "enforcer").Str("query", serialized).Msg("enforced (already has tenant label)")
		return serialized, nil
	}

	// Inject tenant filter if not present
	filter := buildTenantFilter(labelMatch, tenantLabels)
	modified := injectFilter(serialized, filter)

	// Validate modified query by re-parsing
	_, err = traceql.Parse(modified)
	if err != nil {
		return "", fmt.Errorf("failed to inject tenant filter: %w", err)
	}

	log.Trace().Str("function", "enforcer").Str("query", modified).Msg("enforced")
	return modified, nil
}

// buildTenantQuery constructs a minimal TraceQL query with only the tenant label filter.
// For single tenant: { resource.namespace = "tenant1" }
// For multiple tenants: { resource.namespace =~ "tenant1|tenant2|tenant3" }
func buildTenantQuery(labelMatch string, tenantLabels map[string]bool) string {
	operator := "="
	if len(tenantLabels) > 1 {
		operator = "=~"
	}
	return fmt.Sprintf(`{ %s%s"%s" }`,
		labelMatch,
		operator,
		strings.Join(MapKeysToArray(tenantLabels), "|"))
}

// buildTenantFilter creates a tenant label filter expression without the enclosing braces.
// For single tenant: resource.namespace = "tenant1"
// For multiple tenants: resource.namespace =~ "tenant1|tenant2|tenant3"
func buildTenantFilter(labelMatch string, tenantLabels map[string]bool) string {
	operator := "="
	tenantValues := MapKeysToArray(tenantLabels)

	// Only escape for regex operator (multiple tenants)
	if len(tenantLabels) > 1 {
		operator = "=~"
		// Escape regex special characters in tenant values
		escapedValues := make([]string, len(tenantValues))
		for i, v := range tenantValues {
			escapedValues[i] = escapeRegexChars(v)
		}
		return fmt.Sprintf(`%s%s"%s"`,
			labelMatch,
			operator,
			strings.Join(escapedValues, "|"))
	}

	// For single tenant, no escaping needed
	return fmt.Sprintf(`%s%s"%s"`,
		labelMatch,
		operator,
		tenantValues[0])
}

// validateTenantLabels checks if the query contains tenant labels and validates them against allowed values.
// Returns (hasTenantLabel, error) where hasTenantLabel indicates if tenant label was found,
// and error is set if unauthorized tenant label values are found.
func validateTenantLabels(query string, labelMatch string, allowedTenantLabels map[string]bool) (bool, error) {
	// Pattern to match tenant label with value
	// Matches: resource.namespace = "value" or resource.namespace =~ "value1|value2"
	pattern := fmt.Sprintf(`%s\s*=~?\s*[\x60"]([^"\x60]+)[\x60"]`, regexp.QuoteMeta(labelMatch))
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		// No tenant label found, will be injected
		return false, nil
	}

	// Validate all found tenant label values
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value := match[1]

		// Split by pipe for regex patterns
		queryLabels := strings.Split(value, "|")
		for _, queryLabel := range queryLabels {
			queryLabel = strings.TrimSpace(queryLabel)
			if _, ok := allowedTenantLabels[queryLabel]; !ok {
				return true, fmt.Errorf("unauthorized %s: %s", labelMatch, queryLabel)
			}
		}
	}

	return true, nil
}

// injectFilter injects a tenant filter into an existing TraceQL query.
// The filter is combined with the existing query using the AND operator.
func injectFilter(query string, filter string) string {
	trimmed := strings.TrimSpace(query)

	// Handle { true } case (result of empty query parsing)
	if trimmed == "{ true }" {
		return fmt.Sprintf("{ %s }", filter)
	}

	// Remove outer braces, inject filter with AND, add braces back
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inner == "" || inner == "true" {
			return fmt.Sprintf("{ %s }", filter)
		}
		return fmt.Sprintf("{ %s && %s }", filter, inner)
	}

	return query
}

// EnforceMulti modifies a TraceQL query string to enforce multi-label access policy.
// It handles multiple label rules with different operators (=, !=, =~, !~) and logic (AND/OR).
// If the input query is empty, constructs a new query from the policy.
// If the input query is non-empty, validates existing attributes and injects policy filters.
// Returns the modified query or an error if parsing, validation, or modification fails.
func (TraceQLEnforcer) EnforceMulti(query string, policy LabelPolicy) (string, error) {
	log.Trace().Str("function", "enforcer_multi").Str("query", query).Msg("input")

	// Validate policy first
	if err := policy.Validate(); err != nil {
		return "", fmt.Errorf("invalid label policy: %w", err)
	}

	// Handle empty query or just braces
	if query == "" || strings.TrimSpace(query) == "{}" {
		query = buildPolicyQuery(policy)
		log.Trace().Str("function", "enforcer_multi").Str("query", query).Msg("enforcing empty query")
		return query, nil
	}

	log.Trace().Str("function", "enforcer_multi").Str("query", query).Msg("enforcing")

	// Parse query to validate syntax
	ast, err := traceql.Parse(query)
	if err != nil {
		return "", fmt.Errorf("invalid TraceQL syntax: %w", err)
	}

	// Check if it's a no-op query (e.g., "{ true }")
	if ast.IsNoop() {
		query = buildPolicyQuery(policy)
		log.Trace().Str("function", "enforcer_multi").Str("query", query).Msg("enforcing noop query")
		return query, nil
	}

	// Get serialized version for manipulation
	serialized := ast.String()

	// Validate existing attributes against policy
	if err := validatePolicyAttributes(serialized, policy); err != nil {
		return "", err
	}

	// Check if query already contains all policy attributes
	hasPolicyAttributes := checkPolicyAttributes(serialized, policy)
	if hasPolicyAttributes {
		log.Trace().Str("function", "enforcer_multi").Str("query", serialized).Msg("enforced (already has policy attributes)")
		return serialized, nil
	}

	// Inject policy filter if not present
	filter := buildPolicyFilter(policy)
	modified := injectFilter(serialized, filter)

	// Validate modified query by re-parsing
	_, err = traceql.Parse(modified)
	if err != nil {
		return "", fmt.Errorf("failed to inject policy filter: %w", err)
	}

	log.Trace().Str("function", "enforcer_multi").Str("query", modified).Msg("enforced")
	return modified, nil
}

// buildPolicyQuery constructs a minimal TraceQL query from a LabelPolicy.
// Examples:
// - Single rule: { resource.namespace = "prod" }
// - Multiple rules (AND): { resource.namespace = "prod" && resource.team = "backend" }
// - Multiple values: { resource.namespace =~ "prod|staging" }
func buildPolicyQuery(policy LabelPolicy) string {
	filter := buildPolicyFilter(policy)
	return fmt.Sprintf("{ %s }", filter)
}

// buildPolicyFilter creates a filter expression from a LabelPolicy without enclosing braces.
// Handles different operators (=, !=, =~, !~) and logic (AND/OR).
// Examples:
// - resource.namespace = "prod"
// - resource.namespace =~ "prod|staging"
// - resource.namespace = "prod" && resource.team = "backend"
// - resource.namespace = "prod" || resource.team = "backend"
func buildPolicyFilter(policy LabelPolicy) string {
	var filters []string

	for _, rule := range policy.Rules {
		filter := buildRuleFilter(rule)
		filters = append(filters, filter)
	}

	// Combine filters with logic operator
	logicOp := " && "
	if policy.Logic == LogicOR {
		logicOp = " || "
	}

	return strings.Join(filters, logicOp)
}

// buildRuleFilter creates a filter expression for a single LabelRule.
// Handles different operators and multiple values.
// Examples:
// - resource.namespace = "prod"
// - resource.namespace != "test"
// - resource.namespace =~ "prod|staging"
// - resource.team !~ "external|guest"
func buildRuleFilter(rule LabelRule) string {
	operator := rule.Operator

	// For multiple values with = or !=, convert to regex
	if len(rule.Values) > 1 {
		if operator == OperatorEquals {
			operator = OperatorRegexMatch
		} else if operator == OperatorNotEquals {
			operator = OperatorRegexNoMatch
		}
	}

	// Build value expression
	var valueExpr string
	if operator == OperatorRegexMatch || operator == OperatorRegexNoMatch {
		// For regex operators with multiple values, join with |
		if len(rule.Values) > 1 {
			// Escape regex special characters in values
			escapedValues := make([]string, len(rule.Values))
			for i, v := range rule.Values {
				escapedValues[i] = escapeRegexChars(v)
			}
			valueExpr = fmt.Sprintf(`"%s"`, strings.Join(escapedValues, "|"))
		} else {
			// Single value, use as-is (user-provided regex pattern)
			valueExpr = fmt.Sprintf(`"%s"`, rule.Values[0])
		}
	} else {
		// For = and !=, use single value without escaping
		valueExpr = fmt.Sprintf(`"%s"`, rule.Values[0])
	}

	return fmt.Sprintf("%s%s%s", rule.Name, operator, valueExpr)
}

// validatePolicyAttributes validates that any existing policy attributes in the query
// match the allowed values in the policy. Returns error if unauthorized values are found.
func validatePolicyAttributes(query string, policy LabelPolicy) error {
	for _, rule := range policy.Rules {
		// Pattern to match attribute with value
		// Matches: resource.namespace = "value" or resource.namespace =~ "value1|value2"
		pattern := fmt.Sprintf(`%s\s*=~?\s*[\x60"]([^"\x60]+)[\x60"]`, regexp.QuoteMeta(rule.Name))
		re := regexp.MustCompile(pattern)

		matches := re.FindAllStringSubmatch(query, -1)
		if len(matches) == 0 {
			// Attribute not found in query, will be injected
			continue
		}

		// Build allowed values map for quick lookup
		allowedValues := make(map[string]bool)
		for _, value := range rule.Values {
			allowedValues[value] = true
		}

		// Validate all found attribute values
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			value := match[1]

			// Split by pipe for regex patterns
			queryValues := strings.Split(value, "|")
			for _, queryValue := range queryValues {
				queryValue = strings.TrimSpace(queryValue)
				if _, ok := allowedValues[queryValue]; !ok {
					return fmt.Errorf("unauthorized %s: %s", rule.Name, queryValue)
				}
			}
		}
	}

	return nil
}

// checkPolicyAttributes checks if the query already contains all policy attributes.
// Returns true if all attributes from the policy are present in the query.
func checkPolicyAttributes(query string, policy LabelPolicy) bool {
	for _, rule := range policy.Rules {
		// Pattern to match attribute
		pattern := fmt.Sprintf(`%s\s*[=!]=?~?\s*[\x60"]`, regexp.QuoteMeta(rule.Name))
		re := regexp.MustCompile(pattern)

		if !re.MatchString(query) {
			// Attribute not found
			return false
		}
	}

	// All attributes found
	return true
}

// escapeRegexChars escapes special regex characters in a string to prevent regex injection.
// Escapes: . * + ? [ ] ( ) | ^ $ \
func escapeRegexChars(s string) string {
	// Must escape backslash first to avoid double-escaping
	escaped := strings.ReplaceAll(s, `\`, `\\`)

	// Then escape other special characters
	replacements := map[string]string{
		".": `\.`,
		"*": `\*`,
		"+": `\+`,
		"?": `\?`,
		"[": `\[`,
		"]": `\]`,
		"(": `\(`,
		")": `\)`,
		"|": `\|`,
		"^": `\^`,
		"$": `\$`,
	}
	for char, replacement := range replacements {
		escaped = strings.ReplaceAll(escaped, char, replacement)
	}
	return escaped
}
