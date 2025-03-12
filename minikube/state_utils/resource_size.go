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
		result, err := ResourceSizeConverterImpl(val)
		if err != nil {
			panic(err)
		}
		return result
	}
}

func ResourceSizeConverterImpl(val interface{}) (string, error) {
	size, ok := val.(string)
	if !ok {
		return "", errors.New("resource size is not a string")
	}
	sizeMb, err := pkgutil.CalculateSizeInMB(size)
	if err != nil {
		return "", errors.New("invalid resource size value")
	}

	return strconv.Itoa(sizeMb) + "mb", nil
}

func ResourceSizeValidator() schema.SchemaValidateDiagFunc {
	return schema.SchemaValidateDiagFunc(func(val interface{}, path cty.Path) diag.Diagnostics {
		err := ResourceSizeValidatorImpl(val)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil

	})
}

func ResourceSizeValidatorImpl(val interface{}) error {
	size, ok := val.(string)
	if !ok {
		return errors.New("resource size is not a string")
	}
	_, err := pkgutil.CalculateSizeInMB(size)
	if err != nil {
		return errors.New("invalid resource size value")
	}
	return nil
}
