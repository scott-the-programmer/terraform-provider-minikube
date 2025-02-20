package state_utils

import (
	"testing"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/stretchr/testify/assert"
)

func TestGetMemory(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "valid memory size - 2G",
			input:       "2G",
			expected:    2048,
			expectError: false,
		},
		{
			name:        "valid memory size - 1024mb",
			input:       "1024mb",
			expected:    1024,
			expectError: false,
		},
		{
			name:        "no limit case",
			input:       lib.NoLimit,
			expected:    0,
			expectError: false,
		},
		{
			name:        "invalid memory size",
			input:       "invalid",
			expected:    0,
			expectError: true,
		},
		{
			name:        "negative memory size",
			input:       "-1G",
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
			result, err := GetMemory(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
