package state_utils

import (
	"errors"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	pkgutil "k8s.io/minikube/pkg/util"
)

func ResourceSizeConverter() schema.SchemaStateFunc {
	return func(val interface{}) string {
		size, ok := val.(string)
		if !ok {
			panic(errors.New("resource size is not a string"))
		}
		sizeMb, err := pkgutil.CalculateSizeInMB(size)
		if err != nil {
			panic(errors.New("invalid resource size value"))
		}

		return strconv.Itoa(sizeMb) + "mb"
	}
}

func ResourceSizeValidator() schema.SchemaValidateDiagFunc {
	return schema.SchemaValidateDiagFunc(func(val interface{}, path cty.Path) diag.Diagnostics {
		size, ok := val.(string)
		if !ok {
			diag := diag.FromErr(errors.New("resource size is not a string"))
			return diag
		}
		_, err := pkgutil.CalculateSizeInMB(size)
		if err != nil {
			diag := diag.FromErr(errors.New("invalid resource size value"))
			return diag
		}
		return nil
	})
}
