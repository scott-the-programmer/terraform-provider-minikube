//go:build tools

package tools

import (
	// document generation
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"

	// mock
	_ "github.com/golang/mock/mockgen"
)
