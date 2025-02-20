package state_utils

import (
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	pkgutil "k8s.io/minikube/pkg/util"
)

func GetMemory(memoryStr string) (int, error) {
	var memoryMb int
	var err error
	if memoryStr == lib.Max {
		memoryInfo, err := lib.GetMemoryLimit()
		if err != nil {
			return 0, err
		}

		memoryMb = memoryInfo.SystemMemory
	} else if memoryStr == lib.NoLimit {
		memoryMb = 0
	} else {
		err = ResourceSizeValidatorImpl(memoryStr)
		if err != nil {
			return 0, err
		}

		memoryStr, err = ResourceSizeConverterImpl(memoryStr)
		if err != nil {
			return 0, err
		}

		memoryMb, err = pkgutil.CalculateSizeInMB(memoryStr)
		if err != nil {
			return 0, err
		}
	}

	return memoryMb, err
}
