package state_utils

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
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

func CPUConverter() schema.SchemaStateFunc {
	return func(val interface{}) string {
		result, err := CPUConverterImpl(val)
		if err != nil {
			panic(err)
		}
		return result
	}
}

func CPUConverterImpl(val interface{}) (string, error) {
	cpuStr, ok := val.(string)
	if !ok {
		return "", errors.New("cpu value is not a string")
	}

	cpus, err := GetCPUs(cpuStr)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(cpus), nil
}

func CPUValidator() schema.SchemaValidateDiagFunc {
	return schema.SchemaValidateDiagFunc(func(val interface{}, path cty.Path) diag.Diagnostics {
		err := CPUValidatorImpl(val)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	})
}

func CPUValidatorImpl(val interface{}) error {
	cpuStr, ok := val.(string)
	if !ok {
		return errors.New("cpu value is not a string")
	}

	if cpuStr == lib.Max || cpuStr == lib.NoLimit {
		return nil
	}

	cpus, err := strconv.Atoi(cpuStr)
	if err != nil {
		return fmt.Errorf("invalid CPU value: %v", err)
	}

	if cpus <= 0 {
		return fmt.Errorf("CPU count must be positive: %d", cpus)
	}

	return nil
}
