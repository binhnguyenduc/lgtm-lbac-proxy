package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// BenchmarkFileLabelStore_loadLabels_Small benchmarks eager parsing
// with a small number of policies (10 entries)
func BenchmarkFileLabelStore_loadLabels_Small(b *testing.B) {
	yamlContent := `
user1:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod"]
  _logic: AND

user2:
  _rules:
    - name: namespace
      operator: "="
      values: ["staging"]
  _logic: AND

user3:
  _rules:
    - name: namespace
      operator: "="
      values: ["dev"]
  _logic: AND

group1:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod", "staging"]
  _logic: OR

group2:
  _rules:
    - name: namespace
      operator: "=~"
      values: ["backend.*"]
  _logic: AND

group3:
  _rules:
    - name: namespace
      operator: "!="
      values: ["test"]
  _logic: AND

admin:
  _rules:
    - name: '#cluster-wide'
      operator: "="
      values: ["true"]
  _logic: AND

user4:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod", "staging", "dev"]
    - name: team
      operator: "="
      values: ["backend"]
  _logic: AND

user5:
  _rules:
    - name: namespace
      operator: "=~"
      values: ["frontend.*"]
  _logic: AND

group4:
  _rules:
    - name: environment
      operator: "!="
      values: ["archive"]
  _logic: AND
`

	tmpDir := b.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test YAML file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := &FileLabelStore{
			parser:      NewPolicyParser(),
			policyCache: make(map[string]*LabelPolicy),
		}
		v := viper.NewWithOptions(viper.KeyDelimiter("::"))
		_ = store.loadLabels(v, []string{tmpDir})
	}
}

// BenchmarkFileLabelStore_loadLabels_Medium benchmarks eager parsing
// with a medium number of policies (50 entries)
func BenchmarkFileLabelStore_loadLabels_Medium(b *testing.B) {
	// Generate YAML content with 50 entries
	yamlContent := ""
	for i := 1; i <= 50; i++ {
		yamlContent += `
user` + string(rune('0'+i%10)) + `:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod", "staging"]
  _logic: AND
`
	}

	tmpDir := b.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test YAML file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := &FileLabelStore{
			parser:      NewPolicyParser(),
			policyCache: make(map[string]*LabelPolicy),
		}
		v := viper.NewWithOptions(viper.KeyDelimiter("::"))
		_ = store.loadLabels(v, []string{tmpDir})
	}
}

// BenchmarkFileLabelStore_loadLabels_Large benchmarks eager parsing
// with a large number of policies (100 entries)
func BenchmarkFileLabelStore_loadLabels_Large(b *testing.B) {
	// Generate YAML content with 100 entries
	yamlContent := ""
	for i := 1; i <= 100; i++ {
		yamlContent += `
user` + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + `:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod"]
    - name: team
      operator: "=~"
      values: ["backend.*"]
  _logic: AND
`
	}

	tmpDir := b.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test YAML file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := &FileLabelStore{
			parser:      NewPolicyParser(),
			policyCache: make(map[string]*LabelPolicy),
		}
		v := viper.NewWithOptions(viper.KeyDelimiter("::"))
		_ = store.loadLabels(v, []string{tmpDir})
	}
}

// BenchmarkFileLabelStore_GetLabelPolicy_CacheHit benchmarks cache lookup
// performance after eager parsing (cache hit scenario)
func BenchmarkFileLabelStore_GetLabelPolicy_CacheHit(b *testing.B) {
	yamlContent := `
user1:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod"]
  _logic: AND

group1:
  _rules:
    - name: namespace
      operator: "="
      values: ["staging"]
  _logic: AND
`

	tmpDir := b.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test YAML file: %v", err)
	}

	store := &FileLabelStore{
		parser:      NewPolicyParser(),
		policyCache: make(map[string]*LabelPolicy),
	}
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	err = store.loadLabels(v, []string{tmpDir})
	if err != nil {
		b.Fatalf("Failed to load labels: %v", err)
	}

	identity := UserIdentity{
		Username: "user1",
		Groups:   []string{"group1"},
	}

	// First call to populate merged cache
	_, _ = store.GetLabelPolicy(identity, "namespace")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetLabelPolicy(identity, "namespace")
	}
}

// BenchmarkFileLabelStore_GetLabelPolicy_Merge benchmarks policy merging
// performance (cache miss scenario requiring merge)
func BenchmarkFileLabelStore_GetLabelPolicy_Merge(b *testing.B) {
	yamlContent := `
user1:
  _rules:
    - name: namespace
      operator: "="
      values: ["prod"]
  _logic: AND

group1:
  _rules:
    - name: namespace
      operator: "="
      values: ["staging"]
  _logic: AND

group2:
  _rules:
    - name: team
      operator: "="
      values: ["backend"]
  _logic: AND
`

	tmpDir := b.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test YAML file: %v", err)
	}

	store := &FileLabelStore{
		parser:      NewPolicyParser(),
		policyCache: make(map[string]*LabelPolicy),
	}
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	err = store.loadLabels(v, []string{tmpDir})
	if err != nil {
		b.Fatalf("Failed to load labels: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Different group combinations to avoid cache hits
		identity := UserIdentity{
			Username: "user1",
			Groups:   []string{"group1"},
		}
		if i%2 == 0 {
			identity.Groups = []string{"group2"}
		}
		// Clear merged cache to force merge operation
		delete(store.policyCache, "merged:user1:group1")
		delete(store.policyCache, "merged:user1:group2")
		_, _ = store.GetLabelPolicy(identity, "namespace")
	}
}
