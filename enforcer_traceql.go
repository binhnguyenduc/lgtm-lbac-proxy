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
