package state_utils

import (
	"runtime"
	"strconv"
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

func TestCPUConverterImpl(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "valid CPU count",
			input:       "2",
			expected:    "2",
			expectError: false,
		},
		{
			name:        "max CPUs",
			input:       lib.Max,
			expected:    strconv.Itoa(runtime.NumCPU()),
			expectError: false,
		},
		{
			name:        "no limit case",
			input:       lib.NoLimit,
			expected:    "0",
			expectError: false,
		},
		{
			name:        "non-string input",
			input:       42,
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid CPU string",
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CPUConverterImpl(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCPUValidatorImpl(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name:        "valid CPU count",
			input:       "2",
			expectError: false,
		},
		{
			name:        "max CPUs",
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
			input:       42,
			expectError: true,
		},
		{
			name:        "invalid CPU string",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "zero CPU count",
			input:       "0",
			expectError: true,
		},
		{
			name:        "negative CPU count",
			input:       "-1",
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
			err := CPUValidatorImpl(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCPUConverter(t *testing.T) {
	converter := CPUConverter()

	assert.Equal(t, "2", converter("2"))
	assert.Equal(t, strconv.Itoa(runtime.NumCPU()), converter(lib.Max))
	assert.Equal(t, "0", converter(lib.NoLimit))

	assert.Panics(t, func() {
		converter(42) // non-string input should panic
	})
	assert.Panics(t, func() {
		converter("invalid") // unparseable input should panic
	})
}

func TestCPUValidator(t *testing.T) {
	validator := CPUValidator()

	assert.Nil(t, validator("2", nil))
	assert.Nil(t, validator(lib.Max, nil))
	assert.Nil(t, validator(lib.NoLimit, nil))

	assert.NotNil(t, validator(42, nil))        // non-string input
	assert.NotNil(t, validator("invalid", nil)) // unparseable input
	assert.NotNil(t, validator("0", nil))       // zero is not positive
	assert.NotNil(t, validator("", nil))        // empty string
}
