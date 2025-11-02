package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// RawLabelData represents the raw YAML structure from labels.yaml.
// It supports both simple format (map[string]bool) and extended format with _rules.
type RawLabelData map[string]interface{}

// PolicyParser handles parsing and conversion of label configurations
// from YAML format to LabelPolicy structures.
type PolicyParser struct{}

// NewPolicyParser creates a new PolicyParser instance.
func NewPolicyParser() *PolicyParser {
	return &PolicyParser{}
}

// ParseUserPolicy converts raw YAML data for a user/group into a LabelPolicy.
// It detects whether the data uses simple format or extended format and converts accordingly.
//
// Simple format example:
//   namespace: hogarama
//   team: backend
//
// Extended format example:
//   _rules:
//     - name: namespace
//       operator: "="
//       values: ["prod", "staging"]
//   _logic: AND
//
// Mixed format example:
//   namespace: dev  # Auto-converted to rule
//   _rules:
//     - name: environment
//       operator: "!="
//       values: ["production"]
func (p *PolicyParser) ParseUserPolicy(data RawLabelData, defaultLabel string) (*LabelPolicy, error) {
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("empty label data")
	}

	// Check if extended format with _rules key
	if rulesData, hasRules := data["_rules"]; hasRules {
		return p.parseExtendedFormat(data, rulesData)
	}

	// Simple format - convert to policy
	return p.parseSimpleFormat(data, defaultLabel)
}

// parseExtendedFormat handles the new extended YAML format with _rules key.
func (p *PolicyParser) parseExtendedFormat(data RawLabelData, rulesData interface{}) (*LabelPolicy, error) {
	policy := &LabelPolicy{
		Logic: LogicAND, // Default
		Rules: []LabelRule{},
	}

	// Parse _logic if present
	if logicData, ok := data["_logic"]; ok {
		if logic, ok := logicData.(string); ok {
			policy.Logic = logic
		}
	}

	// Parse _override if present
	if overrideData, ok := data["_override"]; ok {
		if override, ok := overrideData.(bool); ok {
			policy.Override = override
		}
	}

	// Parse _rules array
	rulesArray, ok := rulesData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("_rules must be an array")
	}

	for i, ruleData := range rulesArray {
		ruleMap, ok := ruleData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rule %d: must be a map", i)
		}

		rule, err := p.parseRule(ruleMap)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}

		policy.Rules = append(policy.Rules, rule)
	}

	// Also parse any simple format entries alongside _rules (mixed format)
	for key, value := range data {
		if key == "_rules" || key == "_logic" || key == "_override" {
			continue
		}

		// Convert simple entry to rule
		rule, err := p.simpleEntryToRule(key, value)
		if err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Skipping invalid simple format entry in mixed mode")
			continue
		}

		policy.Rules = append(policy.Rules, rule)
	}

	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	return policy, nil
}

// parseSimpleFormat converts the legacy simple format to LabelPolicy.
// Simple format: { "namespace": "value1", "team": "value2" }
// Becomes: Rules with equality operators
func (p *PolicyParser) parseSimpleFormat(data RawLabelData, defaultLabel string) (*LabelPolicy, error) {
	policy := &LabelPolicy{
		Logic:    LogicAND,
		Rules:    []LabelRule{},
		Override: false,
	}

	for key, value := range data {
		rule, err := p.simpleEntryToRule(key, value)
		if err != nil {
			return nil, fmt.Errorf("invalid entry %q: %w", key, err)
		}

		policy.Rules = append(policy.Rules, rule)
	}

	if len(policy.Rules) == 0 {
		return nil, fmt.Errorf("no valid rules found")
	}

	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	return policy, nil
}

// simpleEntryToRule converts a single simple format entry to a LabelRule.
// Handles various value types: string, bool, map[string]bool
func (p *PolicyParser) simpleEntryToRule(key string, value interface{}) (LabelRule, error) {
	rule := LabelRule{
		Name:     key,
		Operator: OperatorEquals,
		Values:   []string{},
	}

	switch v := value.(type) {
	case string:
		// Direct string value: namespace: "prod"
		rule.Values = []string{v}

	case bool:
		// Boolean value: namespace: true (special case for backward compat)
		if v {
			// In old format, map[string]bool where key is the VALUE, not the label name
			// This is handled differently - the key IS the value
			rule.Values = []string{key}
		} else {
			return rule, fmt.Errorf("boolean value must be true")
		}

	case map[string]interface{}:
		// Map format: namespace: { prod: true, staging: true }
		for k, enabled := range v {
			if b, ok := enabled.(bool); ok && b {
				rule.Values = append(rule.Values, k)
			}
		}

	case map[interface{}]interface{}:
		// YAML sometimes parses to map[interface{}]interface{}
		for k, enabled := range v {
			keyStr, ok := k.(string)
			if !ok {
				continue
			}
			if b, ok := enabled.(bool); ok && b {
				rule.Values = append(rule.Values, keyStr)
			}
		}

	default:
		return rule, fmt.Errorf("unsupported value type: %T", value)
	}

	if len(rule.Values) == 0 {
		return rule, fmt.Errorf("no values specified")
	}

	return rule, nil
}

// parseRule converts a map representing a rule into a LabelRule struct.
func (p *PolicyParser) parseRule(ruleMap map[string]interface{}) (LabelRule, error) {
	rule := LabelRule{}

	// Parse name
	name, ok := ruleMap["name"].(string)
	if !ok || name == "" {
		return rule, fmt.Errorf("rule must have a 'name' field")
	}
	rule.Name = name

	// Parse operator
	operator, ok := ruleMap["operator"].(string)
	if !ok || operator == "" {
		return rule, fmt.Errorf("rule must have an 'operator' field")
	}
	rule.Operator = operator

	// Parse values
	valuesData, ok := ruleMap["values"]
	if !ok {
		return rule, fmt.Errorf("rule must have a 'values' field")
	}

	valuesArray, ok := valuesData.([]interface{})
	if !ok {
		return rule, fmt.Errorf("'values' must be an array")
	}

	for i, v := range valuesArray {
		strValue, ok := v.(string)
		if !ok {
			return rule, fmt.Errorf("value %d must be a string", i)
		}
		rule.Values = append(rule.Values, strValue)
	}

	if err := rule.Validate(); err != nil {
		return rule, err
	}

	return rule, nil
}

// IsExtendedFormat checks if the raw data uses the extended format.
// Extended format is identified by the presence of _rules key.
func IsExtendedFormat(data RawLabelData) bool {
	_, hasRules := data["_rules"]
	return hasRules
}
