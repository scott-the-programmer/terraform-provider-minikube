package state_utils

import (
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"runtime"
	"strconv"
	"fmt"
)

func GetCPUs(cpuStr string) (int, error) {
	if cpuStr == lib.Max {
		return runtime.NumCPU(), nil
	} else if cpuStr == lib.NoLimit {
		return 0, nil
	}
	cpus, err := strconv.Atoi(cpuStr)
	if err != nil {
		return 0, err
	}
	if cpus < 0 {
		return 0, fmt.Errorf("CPU count cannot be negative: %d", cpus)
	}
	return cpus, nil
}
