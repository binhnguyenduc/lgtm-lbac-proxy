package main

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPolicyParserParseUserPolicy(t *testing.T) {
	parser := NewPolicyParser()

	tests := []struct {
		name         string
		data         RawLabelData
		defaultLabel string
		wantErr      bool
		wantRules    int
		wantLogic    string
	}{
		{
			name: "extended format single rule",
			data: RawLabelData{
				"_rules": []interface{}{
					map[string]interface{}{
						"name":     "namespace",
						"operator": "=",
						"values":   []interface{}{"prod"},
					},
				},
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    1,
			wantLogic:    LogicAND,
		},
		{
			name: "extended format multiple rules",
			data: RawLabelData{
				"_rules": []interface{}{
					map[string]interface{}{
						"name":     "namespace",
						"operator": "=",
						"values":   []interface{}{"prod", "staging"},
					},
					map[string]interface{}{
						"name":     "team",
						"operator": "=~",
						"values":   []interface{}{"backend.*"},
					},
				},
				"_logic": "AND",
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    2,
			wantLogic:    LogicAND,
		},
		{
			name: "extended format with OR logic",
			data: RawLabelData{
				"_rules": []interface{}{
					map[string]interface{}{
						"name":     "namespace",
						"operator": "=",
						"values":   []interface{}{"prod"},
					},
					map[string]interface{}{
						"name":     "environment",
						"operator": "=",
						"values":   []interface{}{"staging"},
					},
				},
				"_logic": "OR",
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    2,
			wantLogic:    LogicOR,
		},
		{
			name:         "empty data",
			data:         RawLabelData{},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "invalid rule missing name",
			data: RawLabelData{
				"_rules": []interface{}{
					map[string]interface{}{
						"operator": "=",
						"values":   []interface{}{"prod"},
					},
				},
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules is string instead of array (YAML indentation issue)",
			data: RawLabelData{
				"_rules": "cluster",
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules is number instead of array",
			data: RawLabelData{
				"_rules": 42,
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules array contains string instead of map",
			data: RawLabelData{
				"_rules": []interface{}{
					"invalid-string",
				},
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules array contains number instead of map",
			data: RawLabelData{
				"_rules": []interface{}{
					123,
				},
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules array contains nil",
			data: RawLabelData{
				"_rules": []interface{}{
					nil,
				},
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules array has mixed valid and invalid elements",
			data: RawLabelData{
				"_rules": []interface{}{
					map[string]interface{}{
						"name":     "namespace",
						"operator": "=",
						"values":   []interface{}{"prod"},
					},
					"invalid-string",
				},
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
		{
			name: "malformed: _rules is map instead of array",
			data: RawLabelData{
				"_rules": map[string]interface{}{
					"name":     "namespace",
					"operator": "=",
					"values":   []interface{}{"prod"},
				},
			},
			defaultLabel: "namespace",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := parser.ParseUserPolicy(tt.data, tt.defaultLabel)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUserPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if len(policy.Rules) != tt.wantRules {
					t.Errorf("ParseUserPolicy() rules count = %d, want %d", len(policy.Rules), tt.wantRules)
				}
				if policy.Logic != tt.wantLogic {
					t.Errorf("ParseUserPolicy() logic = %s, want %s", policy.Logic, tt.wantLogic)
				}
			}
		})
	}
}

func TestPolicyParserParseRule(t *testing.T) {
	parser := NewPolicyParser()

	tests := []struct {
		name    string
		ruleMap map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid rule",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   []interface{}{"prod"},
			},
			wantErr: false,
		},
		{
			name: "valid regex rule",
			ruleMap: map[string]interface{}{
				"name":     "team",
				"operator": "=~",
				"values":   []interface{}{"backend.*", "frontend.*"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			ruleMap: map[string]interface{}{
				"operator": "=",
				"values":   []interface{}{"prod"},
			},
			wantErr: true,
		},
		{
			name: "missing operator",
			ruleMap: map[string]interface{}{
				"name":   "namespace",
				"values": []interface{}{"prod"},
			},
			wantErr: true,
		},
		{
			name: "missing values",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "<<",
				"values":   []interface{}{"prod"},
			},
			wantErr: true,
		},
		{
			name: "empty values array",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   []interface{}{},
			},
			wantErr: true,
		},
		{
			name: "malformed: values is string instead of array",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   "prod",
			},
			wantErr: true,
		},
		{
			name: "malformed: values is number instead of array",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   42,
			},
			wantErr: true,
		},
		{
			name: "malformed: values array contains number instead of string",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   []interface{}{123},
			},
			wantErr: true,
		},
		{
			name: "malformed: values array contains nil",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   []interface{}{nil},
			},
			wantErr: true,
		},
		{
			name: "malformed: values array has mixed string and non-string",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   []interface{}{"prod", 123},
			},
			wantErr: true,
		},
		{
			name: "malformed: operator is number instead of string",
			ruleMap: map[string]interface{}{
				"name":     "namespace",
				"operator": 123,
				"values":   []interface{}{"prod"},
			},
			wantErr: true,
		},
		{
			name: "malformed: name is number instead of string",
			ruleMap: map[string]interface{}{
				"name":     123,
				"operator": "=",
				"values":   []interface{}{"prod"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.parseRule(tt.ruleMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestYAMLUnmarshallingIssue demonstrates the root cause:
// YAML unmarshalling creates map[interface{}]interface{} instead of map[string]interface{}
// This causes type assertions to fail in parseRule()
func TestYAMLUnmarshallingIssue(t *testing.T) {
	yamlData := `
GrafanaAdmin:
  _logic: AND
  _rules:
    - name: '#cluster-wide'
      operator: '='
      values: ['true']
binh.nguyen@encapital.io:
  _logic: AND
  _rules:
    - name: cluster
      operator: '='
      values: ['prod-dsai', 'prod-monitoring']
`

	// This is how labelstore.go unmarshals the YAML
	var rawData map[string]RawLabelData
	err := yaml.Unmarshal([]byte(yamlData), &rawData)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	parser := NewPolicyParser()

	// Try to parse GrafanaAdmin policy
	grafanaData, ok := rawData["GrafanaAdmin"]
	if !ok {
		t.Fatal("GrafanaAdmin entry not found in rawData")
	}

	// After the fix, this should successfully parse YAML-unmarshalled data
	// The parser now handles both map[string]interface{} and RawLabelData types
	policy, err := parser.ParseUserPolicy(grafanaData, "namespace")
	if err != nil {
		t.Fatalf("Parser should handle YAML-unmarshalled RawLabelData, but got error: %v", err)
	}

	// Verify the policy was parsed correctly
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if len(policy.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(policy.Rules))
	}
	if policy.Rules[0].Name != "#cluster-wide" {
		t.Errorf("Expected rule name '#cluster-wide', got %s", policy.Rules[0].Name)
	}

	// Inspect the actual type of the rule data
	rulesData, hasRules := grafanaData["_rules"]
	if !hasRules {
		t.Fatal("_rules key not found")
	}

	rulesArray, ok := rulesData.([]interface{})
	if !ok {
		t.Fatalf("_rules is not []interface{}, got %T", rulesData)
	}

	if len(rulesArray) == 0 {
		t.Fatal("_rules array is empty")
	}

	// The first rule should be a map, but what type?
	firstRule := rulesArray[0]
	t.Logf("First rule type: %T (value: %v)", firstRule, firstRule)

	// This is the problem:
	// - YAML unmarshaller creates map[interface{}]interface{}
	// - But the code expects map[string]interface{}
	// - Type assertion fails!
	if _, ok := firstRule.(map[string]interface{}); !ok {
		t.Logf("Rule is NOT map[string]interface{}, it's %T - this causes the parser error", firstRule)
	}
}

// TestActualConfiguration tests parsing with the exact real-world configuration that was failing
func TestActualConfiguration(t *testing.T) {
	// This is the exact YAML configuration from the running instance that was causing errors
	yamlData := `
binh.nguyen@encapital.io:
  _logic: AND
  _rules:
    - name: cluster
      operator: =
      values:
        - prod-dsai
        - prod-monitoring

GrafanaAdmin:
  _logic: AND
  _rules:
    - name: '#cluster-wide'
      operator: =
      values:
        - "true"

group1:
  _rules:
    - name: '#cluster-wide'
      operator: '='
      values: ['true']
  _logic: AND

user1:
  _rules:
    - name: namespace
      operator: '='
      values: ['hogarama']
  _logic: AND
`

	// Unmarshal exactly as labelstore.go does
	var rawData map[string]RawLabelData
	err := yaml.Unmarshal([]byte(yamlData), &rawData)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	parser := NewPolicyParser()

	// Test case 1: binh.nguyen@encapital.io with multi-value cluster label
	t.Run("binh.nguyen@encapital.io", func(t *testing.T) {
		userData, ok := rawData["binh.nguyen@encapital.io"]
		if !ok {
			t.Fatal("binh.nguyen@encapital.io entry not found")
		}

		policy, err := parser.ParseUserPolicy(userData, "namespace")
		if err != nil {
			t.Fatalf("Failed to parse binh.nguyen@encapital.io policy: %v", err)
		}

		if policy == nil {
			t.Fatal("Expected non-nil policy")
		}

		// Verify structure
		if len(policy.Rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(policy.Rules))
		}

		if policy.Rules[0].Name != "cluster" {
			t.Errorf("Expected rule name 'cluster', got %s", policy.Rules[0].Name)
		}

		if policy.Rules[0].Operator != OperatorEquals {
			t.Errorf("Expected operator '=', got %s", policy.Rules[0].Operator)
		}

		if len(policy.Rules[0].Values) != 2 {
			t.Errorf("Expected 2 values, got %d: %v", len(policy.Rules[0].Values), policy.Rules[0].Values)
		}

		expectedValues := map[string]bool{"prod-dsai": true, "prod-monitoring": true}
		for _, val := range policy.Rules[0].Values {
			if !expectedValues[val] {
				t.Errorf("Unexpected value: %s", val)
			}
		}

		if policy.Logic != LogicAND {
			t.Errorf("Expected logic 'AND', got %s", policy.Logic)
		}

		t.Logf("✓ binh.nguyen@encapital.io policy parsed correctly: %v", policy)
	})

	// Test case 2: GrafanaAdmin with cluster-wide access
	t.Run("GrafanaAdmin", func(t *testing.T) {
		groupData, ok := rawData["GrafanaAdmin"]
		if !ok {
			t.Fatal("GrafanaAdmin entry not found")
		}

		policy, err := parser.ParseUserPolicy(groupData, "namespace")
		if err != nil {
			t.Fatalf("Failed to parse GrafanaAdmin policy: %v", err)
		}

		if policy == nil {
			t.Fatal("Expected non-nil policy")
		}

		if len(policy.Rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(policy.Rules))
		}

		if policy.Rules[0].Name != "#cluster-wide" {
			t.Errorf("Expected rule name '#cluster-wide', got %s", policy.Rules[0].Name)
		}

		if len(policy.Rules[0].Values) != 1 || policy.Rules[0].Values[0] != "true" {
			t.Errorf("Expected values ['true'], got %v", policy.Rules[0].Values)
		}

		if policy.Logic != LogicAND {
			t.Errorf("Expected logic 'AND', got %s", policy.Logic)
		}

		t.Logf("✓ GrafanaAdmin policy parsed correctly: %v", policy)
	})

	// Test case 3: group1 with cluster-wide (legacy format but valid)
	t.Run("group1", func(t *testing.T) {
		groupData, ok := rawData["group1"]
		if !ok {
			t.Fatal("group1 entry not found")
		}

		policy, err := parser.ParseUserPolicy(groupData, "namespace")
		if err != nil {
			t.Fatalf("Failed to parse group1 policy: %v", err)
		}

		if policy == nil {
			t.Fatal("Expected non-nil policy")
		}

		if !policy.HasClusterWideAccess() {
			t.Error("Expected group1 to have cluster-wide access")
		}

		t.Logf("✓ group1 policy parsed correctly with cluster-wide access")
	})

	// Test case 4: user1 with single namespace
	t.Run("user1", func(t *testing.T) {
		userData, ok := rawData["user1"]
		if !ok {
			t.Fatal("user1 entry not found")
		}

		policy, err := parser.ParseUserPolicy(userData, "namespace")
		if err != nil {
			t.Fatalf("Failed to parse user1 policy: %v", err)
		}

		if policy == nil {
			t.Fatal("Expected non-nil policy")
		}

		if len(policy.Rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(policy.Rules))
		}

		if policy.Rules[0].Name != "namespace" {
			t.Errorf("Expected rule name 'namespace', got %s", policy.Rules[0].Name)
		}

		if len(policy.Rules[0].Values) != 1 || policy.Rules[0].Values[0] != "hogarama" {
			t.Errorf("Expected values ['hogarama'], got %v", policy.Rules[0].Values)
		}

		t.Logf("✓ user1 policy parsed correctly: %v", policy)
	})

	t.Logf("✓✓✓ All actual configuration tests passed!")
}

