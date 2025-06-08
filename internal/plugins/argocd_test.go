package plugins

import (
	"testing"
)

func TestArgocd_ValidateOverrideValues(t *testing.T) {
	argocd := &Argocd{}

	tests := []struct {
		name        string
		overrides   map[string]interface{}
		expectError bool
	}{
		{
			name: "valid override keys",
			overrides: map[string]interface{}{
				"admin.password":  "newpass",
				"server.replicas": 3,
				"redis.enabled":   true,
			},
			expectError: false,
		},
		{
			name: "invalid override key",
			overrides: map[string]interface{}{
				"invalid.key": "value",
			},
			expectError: true,
		},
		{
			name: "mixed valid and invalid keys",
			overrides: map[string]interface{}{
				"admin.password": "newpass",
				"invalid.key":    "value",
			},
			expectError: true,
		},
		{
			name:        "empty overrides",
			overrides:   map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := argocd.ValidateOverrideValues(tt.overrides)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestArgocd_SetOverrideValues(t *testing.T) {
	argocd := &Argocd{
		overrideValues: make(map[string]interface{}),
	}

	testOverrides := map[string]interface{}{
		"admin.password":  "newpass",
		"server.replicas": 3,
		"redis.enabled":   true,
	}

	argocd.SetOverrideValues(testOverrides)

	if len(argocd.overrideValues) != len(testOverrides) {
		t.Errorf("expected %d override values, got %d", len(testOverrides), len(argocd.overrideValues))
	}

	for key, expectedValue := range testOverrides {
		if actualValue, exists := argocd.overrideValues[key]; !exists {
			t.Errorf("expected override value for key %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected override value %v for key %s, got %v", expectedValue, key, actualValue)
		}
	}
}

func TestMergeValues(t *testing.T) {
	tests := []struct {
		name      string
		defaults  map[string]interface{}
		overrides map[string]interface{}
		expected  map[string]interface{}
	}{
		{
			name: "simple merge",
			defaults: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			overrides: map[string]interface{}{
				"key2": "overridden_value2",
				"key3": "value3",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "overridden_value2",
				"key3": "value3",
			},
		},
		{
			name: "nested merge",
			defaults: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 1,
					"port":     8080,
				},
			},
			overrides: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 3,
					"port":     8080,
				},
			},
		},
		{
			name:     "empty defaults",
			defaults: map[string]interface{}{},
			overrides: map[string]interface{}{
				"key1": "value1",
			},
			expected: map[string]interface{}{
				"key1": "value1",
			},
		},
		{
			name: "empty overrides",
			defaults: map[string]interface{}{
				"key1": "value1",
			},
			overrides: map[string]interface{}{},
			expected: map[string]interface{}{
				"key1": "value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeValues(tt.defaults, tt.overrides)

			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetNestedMapValue(t *testing.T) {
	tests := []struct {
		name     string
		initial  map[string]interface{}
		key      string
		value    interface{}
		expected map[string]interface{}
	}{
		{
			name:    "simple key",
			initial: make(map[string]interface{}),
			key:     "key",
			value:   "value",
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:    "nested key",
			initial: make(map[string]interface{}),
			key:     "server.replicas",
			value:   3,
			expected: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 3,
				},
			},
		},
		{
			name: "existing nested structure",
			initial: map[string]interface{}{
				"server": map[string]interface{}{
					"port": 8080,
				},
			},
			key:   "server.replicas",
			value: 3,
			expected: map[string]interface{}{
				"server": map[string]interface{}{
					"port":     8080,
					"replicas": 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setNestedMapValue(tt.initial, tt.key, tt.value)

			if !deepEqual(tt.initial, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.initial)
			}
		})
	}
}

// Helper function to deeply compare maps
func deepEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if bv, ok := b[k]; !ok {
			return false
		} else {
			if !valuesEqual(v, bv) {
				return false
			}
		}
	}

	return true
}

func valuesEqual(a, b interface{}) bool {
	if aMap, ok := a.(map[string]interface{}); ok {
		if bMap, ok := b.(map[string]interface{}); ok {
			return deepEqual(aMap, bMap)
		}
		return false
	}

	return a == b
}
