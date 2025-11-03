package main

import (
	"fmt"
)

// RawLabelData represents the raw YAML structure from labels.yaml.
// Only the extended format with _rules is supported.
type RawLabelData map[string]interface{}

// PolicyParser handles parsing of label configurations
// from YAML format to LabelPolicy structures.
type PolicyParser struct{}

// NewPolicyParser creates a new PolicyParser instance.
func NewPolicyParser() *PolicyParser {
	return &PolicyParser{}
}

// ParseUserPolicy converts raw YAML data for a user/group into a LabelPolicy.
// Only extended format with _rules key is supported.
//
// Extended format example:
//
//	_rules:
//	  - name: namespace
//	    operator: "="
//	    values: ["prod", "staging"]
//	_logic: AND
func (p *PolicyParser) ParseUserPolicy(data RawLabelData, defaultLabel string) (*LabelPolicy, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty label data")
	}

	// Require extended format with _rules key
	rulesData, hasRules := data["_rules"]
	if !hasRules {
		// Check if this looks like simple format (has keys but no _rules)
		if len(data) > 0 && !hasClusterWideOnly(data) {
			return nil, fmt.Errorf("DEPRECATED FORMAT DETECTED: Simple label format is no longer supported in v0.12.0+\n\n" +
				"Migration Required:\n" +
				"  1. Build migration tool: cd cmd/migrate-labels && go build\n" +
				"  2. Validate current format: ./migrate-labels -input labels.yaml -validate\n" +
				"  3. Migrate to extended format: ./migrate-labels -input labels.yaml\n\n" +
				"Required Extended Format Example:\n" +
				"  username:\n" +
				"    _rules:\n" +
				"      - name: namespace\n" +
				"        operator: \"=\"\n" +
				"        values: [\"prod\", \"staging\"]\n" +
				"    _logic: AND\n\n" +
				"Documentation: See cmd/migrate-labels/README.md for detailed migration guide")
		}
		return nil, fmt.Errorf("missing required '_rules' key in label policy")
	}

	return p.parseExtendedFormat(data, rulesData)
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
		// Handle both map[string]interface{} and RawLabelData types
		// RawLabelData is a type alias, so type assertion may fail even though underlying type matches
		var ruleMap map[string]interface{}

		switch v := ruleData.(type) {
		case map[string]interface{}:
			ruleMap = v
		case RawLabelData:
			ruleMap = map[string]interface{}(v)
		default:
			return nil, fmt.Errorf("rule %d: must be a map", i)
		}

		rule, err := p.parseRule(ruleMap)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}

		policy.Rules = append(policy.Rules, rule)
	}

	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	return policy, nil
}

// hasClusterWideOnly checks if the data only contains cluster-wide access
func hasClusterWideOnly(data RawLabelData) bool {
	if len(data) == 0 {
		return false
	}
	for key := range data {
		if key != "#cluster-wide" {
			return false
		}
	}
	return true
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
