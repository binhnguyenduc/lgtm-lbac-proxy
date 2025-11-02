package main

import (
	"testing"
)

func TestLabelRuleValidate(t *testing.T) {
	tests := []struct {
		name    string
		rule    LabelRule
		wantErr bool
	}{
		{
			name: "valid equality rule",
			rule: LabelRule{
				Name:     "namespace",
				Operator: OperatorEquals,
				Values:   []string{"prod"},
			},
			wantErr: false,
		},
		{
			name: "valid regex rule",
			rule: LabelRule{
				Name:     "team",
				Operator: OperatorRegexMatch,
				Values:   []string{"backend.*"},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			rule: LabelRule{
				Name:     "",
				Operator: OperatorEquals,
				Values:   []string{"prod"},
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			rule: LabelRule{
				Name:     "namespace",
				Operator: "<<",
				Values:   []string{"prod"},
			},
			wantErr: true,
		},
		{
			name: "no values",
			rule: LabelRule{
				Name:     "namespace",
				Operator: OperatorEquals,
				Values:   []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid regex pattern",
			rule: LabelRule{
				Name:     "namespace",
				Operator: OperatorRegexMatch,
				Values:   []string{"[invalid"},
			},
			wantErr: true,
		},
		{
			name: "multiple values",
			rule: LabelRule{
				Name:     "namespace",
				Operator: OperatorEquals,
				Values:   []string{"prod", "staging"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LabelRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabelPolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  LabelPolicy
		wantErr bool
	}{
		{
			name: "valid single rule policy",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
				},
				Logic: LogicAND,
			},
			wantErr: false,
		},
		{
			name: "valid multi-rule policy",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
					{Name: "team", Operator: OperatorRegexMatch, Values: []string{"backend.*"}},
				},
				Logic: LogicAND,
			},
			wantErr: false,
		},
		{
			name: "no rules",
			policy: LabelPolicy{
				Rules: []LabelRule{},
				Logic: LogicAND,
			},
			wantErr: true,
		},
		{
			name: "invalid logic",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
				},
				Logic: "XOR",
			},
			wantErr: true,
		},
		{
			name: "default logic to AND",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
				},
				Logic: "",
			},
			wantErr: false,
		},
		{
			name: "OR logic",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
					{Name: "environment", Operator: OperatorEquals, Values: []string{"staging"}},
				},
				Logic: LogicOR,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LabelPolicy.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabelPolicyHasClusterWideAccess(t *testing.T) {
	tests := []struct {
		name   string
		policy LabelPolicy
		want   bool
	}{
		{
			name: "has cluster-wide access",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "#cluster-wide", Operator: OperatorEquals, Values: []string{"true"}},
				},
			},
			want: true,
		},
		{
			name: "no cluster-wide access",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
				},
			},
			want: false,
		},
		{
			name: "cluster-wide among other rules",
			policy: LabelPolicy{
				Rules: []LabelRule{
					{Name: "namespace", Operator: OperatorEquals, Values: []string{"prod"}},
					{Name: "#cluster-wide", Operator: OperatorEquals, Values: []string{"true"}},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.HasClusterWideAccess(); got != tt.want {
				t.Errorf("LabelPolicy.HasClusterWideAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

