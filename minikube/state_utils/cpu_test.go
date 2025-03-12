package state_utils

import (
	"runtime"
	"testing"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/stretchr/testify/assert"
)

func TestGetCPUs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "valid CPU count",
			input:       "2",
			expected:    2,
			expectError: false,
		},
		{
			name:        "max CPUs",
			input:       lib.Max,
			expected:    runtime.NumCPU(),
			expectError: false,
		},
		{
			name:        "no limit case",
			input:       lib.NoLimit,
			expected:    0,
			expectError: false,
		},
		{
			name:        "invalid CPU count",
			input:       "invalid",
			expected:    0,
			expectError: true,
		},
		{
			name:        "negative CPU count",
			input:       "-1",
			expected:    0,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetCPUs(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
