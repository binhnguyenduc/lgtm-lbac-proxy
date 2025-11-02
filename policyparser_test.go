package main

import (
	"testing"
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
			name: "simple format single value",
			data: RawLabelData{
				"namespace": "prod",
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    1,
			wantLogic:    LogicAND,
		},
		{
			name: "simple format boolean",
			data: RawLabelData{
				"hogarama": true,
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    1,
			wantLogic:    LogicAND,
		},
		{
			name: "simple format map",
			data: RawLabelData{
				"namespace": map[string]interface{}{
					"prod":    true,
					"staging": true,
				},
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    1,
			wantLogic:    LogicAND,
		},
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
			name: "mixed format",
			data: RawLabelData{
				"namespace": "dev",
				"_rules": []interface{}{
					map[string]interface{}{
						"name":     "environment",
						"operator": "!=",
						"values":   []interface{}{"production"},
					},
				},
			},
			defaultLabel: "namespace",
			wantErr:      false,
			wantRules:    2, // namespace from simple + environment from rules
			wantLogic:    LogicAND,
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

func TestIsExtendedFormat(t *testing.T) {
	tests := []struct {
		name string
		data RawLabelData
		want bool
	}{
		{
			name: "has _rules key",
			data: RawLabelData{
				"_rules": []interface{}{},
			},
			want: true,
		},
		{
			name: "no _rules key",
			data: RawLabelData{
				"namespace": "prod",
			},
			want: false,
		},
		{
			name: "mixed format",
			data: RawLabelData{
				"namespace": "prod",
				"_rules": []interface{}{
					map[string]interface{}{
						"name":     "team",
						"operator": "=",
						"values":   []interface{}{"backend"},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExtendedFormat(tt.data); got != tt.want {
				t.Errorf("IsExtendedFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolicyParserSimpleEntryToRule(t *testing.T) {
	parser := NewPolicyParser()

	tests := []struct {
		name       string
		key        string
		value      interface{}
		wantErr    bool
		wantValues int
	}{
		{
			name:       "string value",
			key:        "namespace",
			value:      "prod",
			wantErr:    false,
			wantValues: 1,
		},
		{
			name:       "boolean true",
			key:        "hogarama",
			value:      true,
			wantErr:    false,
			wantValues: 1,
		},
		{
			name:    "boolean false",
			key:     "namespace",
			value:   false,
			wantErr: true,
		},
		{
			name: "map with multiple values",
			key:  "namespace",
			value: map[string]interface{}{
				"prod":    true,
				"staging": true,
				"dev":     false,
			},
			wantErr:    false,
			wantValues: 2, // Only true values
		},
		{
			name:    "unsupported type",
			key:     "namespace",
			value:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := parser.simpleEntryToRule(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("simpleEntryToRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if len(rule.Values) != tt.wantValues {
					t.Errorf("simpleEntryToRule() values count = %d, want %d", len(rule.Values), tt.wantValues)
				}
				if rule.Operator != OperatorEquals {
					t.Errorf("simpleEntryToRule() operator = %s, want %s", rule.Operator, OperatorEquals)
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
