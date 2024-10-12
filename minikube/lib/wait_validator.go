package lib

import (
	"fmt"
	"strings"
)

var standardOptions = []string{
	"apiserver",
	"system_pods",
	"default_sa",
	"apps_running",
	"node_ready",
	"kubelet",
}

var specialOptions = []string{
	"all",
	"none",
	"true",
	"false",
}

func ValidateWait(v map[string]bool) error {
	var invalidOptions []string

	for key := range v {
		if !contains(standardOptions, key) || contains(specialOptions, key) {
			invalidOptions = append(invalidOptions, key)
		}
	}

	if len(invalidOptions) > 0 {
		return fmt.Errorf("invalid wait option(s): %s", strings.Join(invalidOptions, ", "))
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ResolveSpecialWaitOptions(input map[string]bool) map[string]bool {
	if input["all"] || input["true"] {
		result := make(map[string]bool)
		for _, opt := range standardOptions {
			result[opt] = true
		}
		return result
	}

	if input["none"] || input["false"] {
		return make(map[string]bool)
	}

	return input
}
