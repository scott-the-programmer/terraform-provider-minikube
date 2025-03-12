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

func TestMemoryConverterImpl(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "valid memory size - 2G",
			input:       "2G",
			expected:    "2048mb",
			expectError: false,
		},
		{
			name:        "valid memory size - 1024mb",
			input:       "1024mb",
			expected:    "1024mb",
			expectError: false,
		},
		{
			name:        "max case",
			input:       lib.Max,
			expected:    lib.Max,
			expectError: false,
		},
		{
			name:        "no limit case",
			input:       lib.NoLimit,
			expected:    lib.NoLimit,
			expectError: false,
		},
		{
			name:        "non-string input",
			input:       123,
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid memory size",
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MemoryConverterImpl(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMemoryValidatorImpl(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name:        "valid memory size - 2G",
			input:       "2G",
			expectError: false,
		},
		{
			name:        "valid memory size - 1024mb",
			input:       "1024mb",
			expectError: false,
		},
		{
			name:        "max case",
			input:       lib.Max,
			expectError: false,
		},
		{
			name:        "no limit case",
			input:       lib.NoLimit,
			expectError: false,
		},
		{
			name:        "non-string input",
			input:       123,
			expectError: true,
		},
		{
			name:        "invalid memory size",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "negative memory size",
			input:       "-1G",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MemoryValidatorImpl(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryConverter(t *testing.T) {
	converter := MemoryConverter()

	// Test normal case
	result := converter("2G")
	assert.Equal(t, "2048mb", result)

	// Test special cases
	assert.Equal(t, lib.Max, converter(lib.Max))
	assert.Equal(t, lib.NoLimit, converter(lib.NoLimit))

	// Test panic case
	assert.Panics(t, func() {
		converter(123) // non-string input should panic
	})
}

func TestMemoryValidator(t *testing.T) {
	validator := MemoryValidator()

	// Test valid cases
	assert.Nil(t, validator("2G", nil))
	assert.Nil(t, validator(lib.Max, nil))
	assert.Nil(t, validator(lib.NoLimit, nil))

	// Test invalid cases
	assert.NotNil(t, validator(123, nil))       // non-string input
	assert.NotNil(t, validator("invalid", nil)) // invalid memory size
	assert.NotNil(t, validator("", nil))        // empty string
}
