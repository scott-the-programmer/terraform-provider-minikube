package state_utils

import (
	"errors"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	pkgutil "k8s.io/minikube/pkg/util"
)

func MemoryConverter() schema.SchemaStateFunc {
	return func(val interface{}) string {
		memory, ok := val.(string)
		if !ok {
			panic(errors.New("memory flag is not a string"))
		}
		memoryMb, err := pkgutil.CalculateSizeInMB(memory)
		if err != nil {
			panic(errors.New("invalid memory value"))
		}

		return strconv.Itoa(memoryMb) + "mb"

	}

}

func MemoryValidator() schema.SchemaValidateDiagFunc {
	return schema.SchemaValidateDiagFunc(func(val interface{}, path cty.Path) diag.Diagnostics {
		memory, ok := val.(string)
		if !ok {
			diag := diag.FromErr(errors.New("memory flag is not a string"))
			return diag
		}
		_, err := pkgutil.CalculateSizeInMB(memory)
		if err != nil {
			diag := diag.FromErr(errors.New("invalid memory value"))
			return diag
		}
		return nil
	})
}
