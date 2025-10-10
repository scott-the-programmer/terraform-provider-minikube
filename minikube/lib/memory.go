package lib

import (
	"k8s.io/minikube/pkg/minikube/machine"
)

var NoLimit = "no-limit"
var Max = "max"

// MemoryInfo holds system and container memory information
type MemoryInfo struct {
	SystemMemory int
}

// GetMemoryLimits returns the amount of memory allocated to the system and container
// The return values are in MiB
func GetMemoryLimit() (*MemoryInfo, error) {
	info, _, memErr, _ := machine.LocalHostInfo()

	if memErr != nil {
		return nil, memErr
	}

	// Subtract 1gb for overhead
	memInfo := &MemoryInfo{
		SystemMemory: int(info.Memory) - 1024,
	}

	return memInfo, nil
}
