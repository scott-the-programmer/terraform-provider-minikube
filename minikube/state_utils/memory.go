package state_utils

import (
	"errors"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func MemoryConverter() schema.SchemaStateFunc {
	return func(val interface{}) string {
		result, err := MemoryConverterImpl(val)
		if err != nil {
			panic(err)
		}
		return result
	}
}

func MemoryConverterImpl(val interface{}) (string, error) {
	memoryStr, ok := val.(string)
	if !ok {
		return "", errors.New("memory value is not a string")
	}

	if memoryStr == lib.Max || memoryStr == lib.NoLimit {
		return memoryStr, nil
	}

	return ResourceSizeConverterImpl(memoryStr)
}

func MemoryValidator() schema.SchemaValidateDiagFunc {
	return schema.SchemaValidateDiagFunc(func(val interface{}, path cty.Path) diag.Diagnostics {
		err := MemoryValidatorImpl(val)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	})
}

func MemoryValidatorImpl(val interface{}) error {
	memoryStr, ok := val.(string)
	if !ok {
		return errors.New("memory value is not a string")
	}

	if memoryStr == lib.Max || memoryStr == lib.NoLimit {
		return nil
	}

	return ResourceSizeValidatorImpl(memoryStr)
}
