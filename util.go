package main

import (
	"sort"
	"strings"
)

func ContainsIgnoreCase(s []string, e string) bool {
	for _, v := range s {
		if strings.EqualFold(v, e) {
			return true
		}
	}
	return false
}

func MapKeysToArray[K comparable, V any](tenantLabel map[K]V) []K {
	tenantLabelKeys := make([]K, 0, len(tenantLabel))
	for key := range tenantLabel {
		tenantLabelKeys = append(tenantLabelKeys, key)
	}

	// Sort if keys are strings for deterministic output
	if len(tenantLabelKeys) > 0 {
		if keys, ok := any(tenantLabelKeys).([]string); ok {
			sort.Strings(keys)
			return any(keys).([]K)
		}
	}

	return tenantLabelKeys
}
