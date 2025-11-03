package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestFileLabelStoreLoadLabelsPreserveCase tests that case sensitivity is preserved
// when loading labels from YAML files
func TestFileLabelStoreLoadLabelsPreserveCase(t *testing.T) {
	tests := []struct {
		name             string
		yamlContent      string
		expectedKeys     []string
		unexpectedKeys   []string
		description      string
	}{
		{
			name: "mixed case username",
			yamlContent: `
GrafanaAdmin:
  _logic: AND
  _rules:
    - name: '#cluster-wide'
      operator: =
      values: ["true"]
`,
			expectedKeys:   []string{"GrafanaAdmin"},
			unexpectedKeys: []string{"grafanaadmin", "GRAFANAADMIN"},
			description:    "GrafanaAdmin should preserve exact case, not be lowercased to grafanaadmin",
		},
		{
			name: "email-style username with mixed case",
			yamlContent: `
BiNh.NgUyEn@EnCapital.io:
  _logic: AND
  _rules:
    - name: cluster
      operator: =
      values: ["prod"]
`,
			expectedKeys:   []string{"BiNh.NgUyEn@EnCapital.io"},
			unexpectedKeys: []string{"binh.nguyen@encapital.io"},
			description:    "Email-style usernames should preserve original case",
		},
		{
			name: "multiple mixed case groups",
			yamlContent: `
BackendTeam:
  _logic: AND
  _rules:
    - name: namespace
      operator: =
      values: ["prod"]

FrontendTeam:
  _logic: OR
  _rules:
    - name: namespace
      operator: =
      values: ["staging"]

admin-group:
  _logic: AND
  _rules:
    - name: '#cluster-wide'
      operator: =
      values: ["true"]
`,
			expectedKeys:   []string{"BackendTeam", "FrontendTeam", "admin-group"},
			unexpectedKeys: []string{"backendteam", "frontendteam", "ADMIN-GROUP"},
			description:    "All group names should preserve their exact case",
		},
		{
			name: "uppercase and lowercase variations",
			yamlContent: `
ALLCAPS:
  _logic: AND
  _rules:
    - name: env
      operator: =
      values: ["prod"]

lowercase:
  _logic: AND
  _rules:
    - name: env
      operator: =
      values: ["dev"]

MixedCase:
  _logic: AND
  _rules:
    - name: env
      operator: =
      values: ["staging"]
`,
			expectedKeys:   []string{"ALLCAPS", "lowercase", "MixedCase"},
			unexpectedKeys: []string{"allcaps", "LOWERCASE", "mixedcase"},
			description:    "All case variations should be preserved exactly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tmpDir := t.TempDir()
			yamlFile := filepath.Join(tmpDir, "labels.yaml")
			err := os.WriteFile(yamlFile, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test YAML file: %v", err)
			}

			// Create FileLabelStore and load labels
			store := &FileLabelStore{}
			v := viper.NewWithOptions(viper.KeyDelimiter("::"))
			err = store.loadLabels(v, []string{tmpDir})
			if err != nil {
				t.Fatalf("Failed to load labels: %v", err)
			}

			// Verify expected keys are present in parsed policy cache
			for _, expectedKey := range tt.expectedKeys {
				cacheKey := "entry:" + expectedKey
				if _, ok := store.policyCache[cacheKey]; !ok {
					t.Errorf("Expected key %q not found in policyCache. Available keys: %v", expectedKey, mapPolicyCacheKeys(store.policyCache))
				}
			}

			// Verify unexpected keys are not present (i.e., case was lowercased)
			for _, unexpectedKey := range tt.unexpectedKeys {
				cacheKey := "entry:" + unexpectedKey
				if _, ok := store.policyCache[cacheKey]; ok {
					t.Errorf("Unexpected key %q found in policyCache (case was incorrectly modified). %s", unexpectedKey, tt.description)
				}
			}
		})
	}
}


// TestFileLabelStoreMultipleUsersAndGroupsPreserveCase tests that case sensitivity
// is preserved when loading YAML with multiple users and groups of varying case
func TestFileLabelStoreMultipleUsersAndGroupsPreserveCase(t *testing.T) {
	yamlContent := `
GrafanaAdmin:
  _logic: AND
  _rules:
    - name: '#cluster-wide'
      operator: =
      values: ["true"]

BackendTeam:
  _logic: AND
  _rules:
    - name: namespace
      operator: =
      values: ["prod"]

FrontendTeam:
  _logic: OR
  _rules:
    - name: namespace
      operator: =
      values: ["staging", "dev"]

john.doe@example.com:
  _logic: AND
  _rules:
    - name: env
      operator: =
      values: ["sandbox"]
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	store := &FileLabelStore{}
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	err = store.loadLabels(v, []string{tmpDir})
	if err != nil {
		t.Fatalf("Failed to load labels: %v", err)
	}

	// Verify all keys are present with correct case (main goal of the fix)
	expectedKeys := map[string]bool{
		"GrafanaAdmin":           true,
		"BackendTeam":            true,
		"FrontendTeam":           true,
		"john.doe@example.com": true,
	}

	for key := range expectedKeys {
		cacheKey := "entry:" + key
		if _, ok := store.policyCache[cacheKey]; !ok {
			t.Errorf("Expected key %q not found with correct case. Available keys: %v. "+
				"This indicates case sensitivity was not preserved from YAML file.", key, mapPolicyCacheKeys(store.policyCache))
		}
	}

	// Verify lowercased variations are NOT present (would happen with old Viper behavior)
	unexpectedLowercaseKeys := []string{
		"grafanaadmin",
		"backendteam",
		"frontendteam",
		"john.doe@EXAMPLE.COM", // Different case
	}

	for _, key := range unexpectedLowercaseKeys {
		cacheKey := "entry:" + key
		if _, ok := store.policyCache[cacheKey]; ok {
			t.Errorf("Unexpected key %q found in policyCache. Case was incorrectly modified "+
				"(old Viper behavior that we fixed).", key)
		}
	}
}

// TestFileLabelStoreDirectYAMLParsing tests that direct YAML parsing preserves case
// correctly. This tests the new implementation that reads YAML files directly instead
// of using Viper, which normalizes all keys to lowercase.
func TestFileLabelStoreDirectYAMLParsing(t *testing.T) {
	yamlContent := `
testUser123:
  _logic: AND
  _rules:
    - name: namespace
      operator: =
      values: ["test"]

TestUser456:
  _logic: AND
  _rules:
    - name: namespace
      operator: =
      values: ["test"]

TESTUSER789:
  _logic: AND
  _rules:
    - name: namespace
      operator: =
      values: ["test"]
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "labels.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	// Test with direct YAML parsing (current implementation)
	store := &FileLabelStore{}
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	err = store.loadLabels(v, []string{tmpDir})
	if err != nil {
		t.Fatalf("Failed to load labels: %v", err)
	}

	// Verify all variations are present with correct case in parsed policy cache
	if _, ok := store.policyCache["entry:testUser123"]; !ok {
		t.Error("Expected key 'testUser123' not found in direct YAML parsing result")
	}
	if _, ok := store.policyCache["entry:TestUser456"]; !ok {
		t.Error("Expected key 'TestUser456' not found in direct YAML parsing result")
	}
	if _, ok := store.policyCache["entry:TESTUSER789"]; !ok {
		t.Error("Expected key 'TESTUSER789' not found in direct YAML parsing result")
	}

	// Verify that lowercased versions are NOT present (which would happen with Viper)
	if _, ok := store.policyCache["entry:testuser456"]; ok {
		t.Error("Unexpected lowercased key 'testuser456' found (indicates case was modified by Viper)")
	}
	if _, ok := store.policyCache["entry:testuser789"]; ok {
		t.Error("Unexpected lowercased key 'testuser789' found (indicates case was modified by Viper)")
	}
}

// Helper function to get policy cache keys for debugging
func mapPolicyCacheKeys(m map[string]*LabelPolicy) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
