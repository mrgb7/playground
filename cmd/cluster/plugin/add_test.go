package plugin

import (
	"testing"
)

func TestParseSetValues(t *testing.T) {
	tests := []struct {
		name        string
		setValues   []string
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name:      "simple key-value",
			setValues: []string{"admin.password=newpass"},
			expected: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "newpass",
				},
			},
			expectError: false,
		},
		{
			name:      "multiple values",
			setValues: []string{"admin.password=newpass", "server.replicas=3"},
			expected: map[string]interface{}{
				"admin": map[string]interface{}{
					"password": "newpass",
				},
				"server": map[string]interface{}{
					"replicas": 3,
				},
			},
			expectError: false,
		},
		{
			name:      "boolean values",
			setValues: []string{"redis.enabled=true", "dex.enabled=false"},
			expected: map[string]interface{}{
				"redis": map[string]interface{}{
					"enabled": true,
				},
				"dex": map[string]interface{}{
					"enabled": false,
				},
			},
			expectError: false,
		},
		{
			name:      "float values",
			setValues: []string{"resources.cpu=1.5"},
			expected: map[string]interface{}{
				"resources": map[string]interface{}{
					"cpu": 1.5,
				},
			},
			expectError: false,
		},
		{
			name:        "invalid format",
			setValues:   []string{"invalid-format"},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty key",
			setValues:   []string{"=value"},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSetValues(tt.setValues)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if !mapsEqual(result, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"string value", "hello", "hello"},
		{"boolean true", "true", true},
		{"boolean false", "false", false},
		{"integer", "123", 123},
		{"float", "123.45", 123.45},
		{"string that looks like number", "123abc", "123abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseValue(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected map[string]interface{}
	}{
		{
			name:  "simple key",
			key:   "key",
			value: "value",
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "nested key",
			key:   "server.replicas",
			value: 3,
			expected: map[string]interface{}{
				"server": map[string]interface{}{
					"replicas": 3,
				},
			},
		},
		{
			name:  "deep nested key",
			key:   "server.resources.requests.cpu",
			value: "100m",
			expected: map[string]interface{}{
				"server": map[string]interface{}{
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu": "100m",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]interface{})
			setNestedValue(result, tt.key, tt.value)

			if !mapsEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helper function to deeply compare maps
func mapsEqual(a, b map[string]interface{}) bool {
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
			return mapsEqual(aMap, bMap)
		}
		return false
	}

	return a == b
}
