package store

import "testing"

func TestFilterStr(t *testing.T) {
	tests := []struct {
		name     string
		filters  map[string]any
		key      string
		expected *string
	}{
		{
			name:     "key exists with string value",
			filters:  map[string]any{"certType": "ca"},
			key:      "certType",
			expected: strPtr("ca"),
		},
		{
			name:     "key does not exist",
			filters:  map[string]any{"certType": "ca"},
			key:      "missing",
			expected: nil,
		},
		{
			name:     "key exists with non-string value",
			filters:  map[string]any{"count": 42},
			key:      "count",
			expected: nil,
		},
		{
			name:     "empty filters",
			filters:  map[string]any{},
			key:      "certType",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterStr(tt.filters, tt.key)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %q", *result)
				}
			} else {
				if result == nil {
					t.Errorf("expected %q, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("expected %q, got %q", *tt.expected, *result)
				}
			}
		})
	}
}

func strPtr(s string) *string { return &s }
