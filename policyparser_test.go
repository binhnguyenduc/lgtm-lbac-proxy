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
