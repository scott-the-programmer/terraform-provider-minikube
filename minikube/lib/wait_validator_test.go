package lib

import (
	"reflect"
	"testing"
)

func TestValidateWait(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]bool
		expectedError string
	}{
		{
			name:          "Valid options",
			input:         map[string]bool{"apiserver": true, "system_pods": true},
			expectedError: "",
		},
		{
			name:          "Invalid option",
			input:         map[string]bool{"invalid_option": true},
			expectedError: "invalid wait option(s): invalid_option",
		},
		{
			name:          "Multiple invalid options",
			input:         map[string]bool{"invalid1": true, "invalid2": true, "apiserver": true},
			expectedError: "invalid wait option(s): invalid1, invalid2",
		},
		{
			name:          "Empty input",
			input:         map[string]bool{},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWait(tt.input)
			if (err == nil && tt.expectedError != "") || (err != nil && err.Error() != tt.expectedError) {
				t.Errorf("ValidateWait() error = %v, expectedError %v", err, tt.expectedError)
			}
		})
	}
}

func TestResolveSpecialWaitOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]bool
		expected map[string]bool
	}{
		{
			name:     "All true",
			input:    map[string]bool{"all": true},
			expected: map[string]bool{"apiserver": true, "system_pods": true, "default_sa": true, "apps_running": true, "node_ready": true, "kubelet": true},
		},
		{
			name:     "True",
			input:    map[string]bool{"true": true},
			expected: map[string]bool{"apiserver": true, "system_pods": true, "default_sa": true, "apps_running": true, "node_ready": true, "kubelet": true},
		},
		{
			name:     "None",
			input:    map[string]bool{"none": true},
			expected: map[string]bool{},
		},
		{
			name:     "False",
			input:    map[string]bool{"false": true},
			expected: map[string]bool{},
		},
		{
			name:     "Standard options",
			input:    map[string]bool{"apiserver": true, "system_pods": true},
			expected: map[string]bool{"apiserver": true, "system_pods": true},
		},
		{
			name:     "Empty input",
			input:    map[string]bool{},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveSpecialWaitOptions(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ResolveSpecialWaitOptions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "Item present",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "Item not present",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "Empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}
