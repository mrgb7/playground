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
				"admin": map[string]interface{}{
					"password": "newpass",
				},
			},
			expectError: false,
		},
		{
			name: "invalid override key",
			overrides: map[string]interface{}{
				"invalid": map[string]interface{}{
					"key": "value",
				},
			},
			expectError: true,
		},
		{
			name: "mixed valid and invalid keys",
			overrides: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "newpass",
				},
				"invalid": map[string]interface{}{
					"key": "value",
				},
			},
			expectError: true,
		},
		{
			name:        "empty overrides",
			overrides:   map[string]interface{}{},
			expectError: false,
		},
		{
			name: "deeply nested invalid key",
			overrides: map[string]interface{}{
				"server": map[string]interface{}{
					"invalid": map[string]interface{}{
						"deep": "value",
					},
				},
			},
			expectError: true,
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

func TestFlattenKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected []string
	}{
		{
			name: "simple nested structure",
			input: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "secret",
				},
			},
			expected: []string{"admin.password"},
		},
		{
			name: "multiple nested keys",
			input: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "secret",
				},
				"server": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: []string{"admin.password", "server.replicas"},
		},
		{
			name: "deeply nested structure",
			input: map[string]interface{}{
				"server": map[string]interface{}{
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu": "100m",
						},
					},
				},
			},
			expected: []string{"server.resources.requests.cpu"},
		},
		{
			name: "mixed flat and nested",
			input: map[string]interface{}{
				"enabled": true,
				"server": map[string]interface{}{
					"port": 8080,
				},
			},
			expected: []string{"enabled", "server.port"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenKeys(tt.input, "")

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			// Convert to map for easier comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, key := range result {
				resultMap[key] = true
			}

			for _, expectedKey := range tt.expected {
				if !resultMap[expectedKey] {
					t.Errorf("expected key '%s' not found in result: %v", expectedKey, result)
				}
			}
		})
	}
}

func TestThreeWayMerge(t *testing.T) {
	tests := []struct {
		name      string
		defaults  map[string]interface{}
		current   map[string]interface{}
		overrides map[string]interface{}
		expected  map[string]interface{}
	}{
		{
			name: "three-way merge preserves current modifications",
			defaults: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 1,
					"port":     8080,
				},
				"redis": map[string]interface{}{
					"enabled": false,
				},
			},
			current: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 1,
					"port":     8080,
					"ingress": map[string]interface{}{
						"enabled": true,
						"host":    "argocd.local",
					},
				},
				"redis": map[string]interface{}{
					"enabled": false,
				},
				"tls": map[string]interface{}{
					"enabled":     true,
					"certificate": "auto",
				},
			},
			overrides: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "newsecret",
				},
				"server": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "newsecret",
				},
				"server": map[string]interface{}{
					"replicas": 3, // Override takes precedence
					"port":     8080,
					"ingress": map[string]interface{}{
						"enabled": true,
						"host":    "argocd.local",
					},
				},
				"redis": map[string]interface{}{
					"enabled": false,
				},
				"tls": map[string]interface{}{
					"enabled":     true,
					"certificate": "auto",
				},
			},
		},
		{
			name: "override takes precedence over current",
			defaults: map[string]interface{}{
				"setting": "default",
			},
			current: map[string]interface{}{
				"setting": "current_modified",
			},
			overrides: map[string]interface{}{
				"setting": "user_override",
			},
			expected: map[string]interface{}{
				"setting": "user_override",
			},
		},
		{
			name: "current preserves additional fields not in defaults",
			defaults: map[string]interface{}{
				"basic": "setting",
			},
			current: map[string]interface{}{
				"basic":   "setting",
				"plugin1": "added_by_plugin",
				"plugin2": "also_added",
			},
			overrides: map[string]interface{}{
				"user": "override",
			},
			expected: map[string]interface{}{
				"basic":   "setting",
				"plugin1": "added_by_plugin",
				"plugin2": "also_added",
				"user":    "override",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate three-way merge: defaults -> current -> overrides
			step1 := mergeValues(tt.defaults, tt.current)
			result := mergeValues(step1, tt.overrides)

			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
