package cosmosutils

import (
	"testing"
)

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		{
			v1:       "1.2.3",
			v2:       "1.2.2",
			expected: true,
		},
		{
			v1:       "2.0.0",
			v2:       "1.9.9",
			expected: true,
		},
		{
			v1:       "1.0.0",
			v2:       "1.0.0",
			expected: false,
		},
		{
			v1:       "1.0.0",
			v2:       "1.0.0-1",
			expected: false,
		},
		{
			v1:       "1.0.0-2",
			v2:       "1.0.0-1",
			expected: true,
		},
		{
			v1:       "1.2.0",
			v2:       "1.1.9",
			expected: true,
		},
		{
			v1:       "2.0.0",
			v2:       "1.9.9",
			expected: true,
		},
		{
			v1:       "1.0.0",
			v2:       "2.0.0",
			expected: false,
		},
		{
			v1:       "1.0.0",
			v2:       "1.0.0-2",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSemVer(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareSemVer(%s, %s) = %v, want %v",
					tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}
