//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

package main

import (
	"flag"
	"os"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/worker"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	// Worker dispatch must precede plugin serving and, in particular, flag
	// parsing: the child is a re-executed provider binary whose readable
	// contract is one JSON request on stdin and one framed response on stdout.
	// Package-level init of the embedded Minikube libraries has already run by
	// this point (PATH pre-pending and log registration); those effects are
	// inert for workers and re-execute on every fresh worker.
	if worker.IsWorkerChild() {
		os.Exit(worker.Serve(os.Stdin, os.Stdout))
	}

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
