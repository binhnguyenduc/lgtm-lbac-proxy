package main

import (
	"testing"
)

// Benchmark policy parsing performance
func BenchmarkPolicyParser_SimpleFormat(b *testing.B) {
	parser := NewPolicyParser()
	data := RawLabelData{
		"namespace": "prod",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseUserPolicy(data, "namespace")
	}
}

func BenchmarkPolicyParser_ExtendedFormat(b *testing.B) {
	parser := NewPolicyParser()
	data := RawLabelData{
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
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseUserPolicy(data, "namespace")
	}
}

func BenchmarkPolicyParser_MixedFormat(b *testing.B) {
	parser := NewPolicyParser()
	data := RawLabelData{
		"namespace": "dev",
		"_rules": []interface{}{
			map[string]interface{}{
				"name":     "environment",
				"operator": "!=",
				"values":   []interface{}{"production"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseUserPolicy(data, "namespace")
	}
}

// Benchmark PromQL enforcement
func BenchmarkPromQLEnforcer_SingleLabel(b *testing.B) {
	enforcer := PromQLEnforcer{}
	query := `rate(http_requests_total[5m])`
	labels := map[string]bool{"prod": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.Enforce(query, labels, "namespace")
	}
}

func BenchmarkPromQLEnforcer_MultiLabel(b *testing.B) {
	enforcer := PromQLEnforcer{}
	query := `rate(http_requests_total[5m])`
	policy := LabelPolicy{
		Rules: []LabelRule{
			{
				Name:     "namespace",
				Operator: "=",
				Values:   []string{"prod", "staging"},
			},
			{
				Name:     "team",
				Operator: "=~",
				Values:   []string{"backend.*"},
			},
		},
		Logic: LogicAND,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.EnforceMulti(query, policy)
	}
}

// Benchmark LogQL enforcement
func BenchmarkLogQLEnforcer_SingleLabel(b *testing.B) {
	enforcer := LogQLEnforcer{}
	query := `{job="app"} |= "error"`
	labels := map[string]bool{"prod": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.Enforce(query, labels, "namespace")
	}
}

func BenchmarkLogQLEnforcer_MultiLabel(b *testing.B) {
	enforcer := LogQLEnforcer{}
	query := `{job="app"} |= "error"`
	policy := LabelPolicy{
		Rules: []LabelRule{
			{
				Name:     "namespace",
				Operator: "=",
				Values:   []string{"prod", "staging"},
			},
			{
				Name:     "environment",
				Operator: "!=",
				Values:   []string{"test"},
			},
		},
		Logic: LogicAND,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.EnforceMulti(query, policy)
	}
}

// Benchmark TraceQL enforcement
func BenchmarkTraceQLEnforcer_SingleLabel(b *testing.B) {
	enforcer := TraceQLEnforcer{}
	query := `{ span.http.status_code = 500 }`
	labels := map[string]bool{"prod": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.Enforce(query, labels, "resource.namespace")
	}
}

func BenchmarkTraceQLEnforcer_MultiLabel(b *testing.B) {
	enforcer := TraceQLEnforcer{}
	query := `{ span.http.status_code = 500 }`
	policy := LabelPolicy{
		Rules: []LabelRule{
			{
				Name:     "resource.namespace",
				Operator: "=",
				Values:   []string{"prod", "staging"},
			},
			{
				Name:     "resource.team",
				Operator: "=~",
				Values:   []string{"backend.*"},
			},
		},
		Logic: LogicAND,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.EnforceMulti(query, policy)
	}
}

// Benchmark label rule validation
func BenchmarkLabelRule_Validate(b *testing.B) {
	rule := LabelRule{
		Name:     "namespace",
		Operator: "=",
		Values:   []string{"prod", "staging", "dev"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rule.Validate()
	}
}

func BenchmarkLabelPolicy_Validate(b *testing.B) {
	policy := LabelPolicy{
		Rules: []LabelRule{
			{
				Name:     "namespace",
				Operator: "=",
				Values:   []string{"prod", "staging"},
			},
			{
				Name:     "team",
				Operator: "=~",
				Values:   []string{"backend.*", "frontend.*"},
			},
			{
				Name:     "environment",
				Operator: "!=",
				Values:   []string{"test"},
			},
		},
		Logic: LogicAND,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.Validate()
	}
}

// Benchmark complex scenarios
func BenchmarkPromQLEnforcer_ComplexQuery(b *testing.B) {
	enforcer := PromQLEnforcer{}
	query := `sum(rate(http_requests_total{status=~"5.."}[5m])) by (handler) / sum(rate(http_requests_total[5m])) by (handler)`
	policy := LabelPolicy{
		Rules: []LabelRule{
			{
				Name:     "namespace",
				Operator: "=~",
				Values:   []string{"prod|staging|dev"},
			},
			{
				Name:     "team",
				Operator: "!=",
				Values:   []string{"external"},
			},
			{
				Name:     "environment",
				Operator: "=",
				Values:   []string{"production"},
			},
		},
		Logic: LogicAND,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enforcer.EnforceMulti(query, policy)
	}
}

func BenchmarkPolicyParser_LargeRuleSet(b *testing.B) {
	parser := NewPolicyParser()
	// Simulate a user with many label rules
	data := RawLabelData{
		"_rules": []interface{}{
			map[string]interface{}{
				"name":     "namespace",
				"operator": "=",
				"values":   []interface{}{"prod", "staging", "dev", "test"},
			},
			map[string]interface{}{
				"name":     "team",
				"operator": "=~",
				"values":   []interface{}{"backend.*", "frontend.*", "platform.*"},
			},
			map[string]interface{}{
				"name":     "environment",
				"operator": "!=",
				"values":   []interface{}{"archive", "deprecated"},
			},
			map[string]interface{}{
				"name":     "region",
				"operator": "=",
				"values":   []interface{}{"us-east-1", "us-west-2", "eu-central-1"},
			},
			map[string]interface{}{
				"name":     "cluster",
				"operator": "=~",
				"values":   []interface{}{"k8s-.*"},
			},
		},
		"_logic": "AND",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseUserPolicy(data, "namespace")
	}
}
