package state_utils

import (
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"runtime"
	"strconv"
)

func GetCPUs(cpuStr string) (int, error) {
	if cpuStr == lib.Max {
		return runtime.NumCPU(), nil
	} else if cpuStr == lib.NoLimit {
		return 0, nil
	}
	return strconv.Atoi(cpuStr)
}
