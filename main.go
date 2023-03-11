//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

package main

import (
	"flag"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		Debug:        debug,
		ProviderAddr: "registry.terraform.io/hashicorp/minikube",
		ProviderFunc: minikube.Provider,
	}

	plugin.Serve(opts)
}
