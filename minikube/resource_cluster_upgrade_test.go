package minikube

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestAccClusterUpgrade applies exactly the same configuration and state first
// with the latest registry release, then with the provider from this checkout.
func TestAccClusterUpgrade(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to create a real Docker cluster")
	}

	// Reattach the branch provider at the same address as the published provider;
	// otherwise the SDK defaults to the hashicorp namespace.
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "scott-the-programmer")
	version := ""
	if from := os.Getenv("UPGRADE_FROM_VERSION"); from != "" {
		version = fmt.Sprintf("version = %q", "= "+from)
	}
	config := fmt.Sprintf(`
terraform {
  required_providers {
    minikube = {
      source = "scott-the-programmer/minikube"
      %s
    }
  }
}

resource "minikube_cluster" "this" {
  cluster_name       = "tf-minikube-upgrade-%d"
  driver             = "docker"
  vm                 = false
  nodes              = 1
  cpus               = "2"
  memory             = "4g"
  kubernetes_version = "v1.34.0"
  container_runtime  = "docker"
  addons             = ["default-storageclass", "storage-provisioner"]
  delete_on_failure  = true
}

output "host" {
  value = minikube_cluster.this.host
}

output "client_certificate" {
  sensitive = true
  value     = minikube_cluster.this.client_certificate
}

output "client_key" {
  sensitive = true
  value     = minikube_cluster.this.client_key
}

output "cluster_ca_certificate" {
  sensitive = true
  value     = minikube_cluster.this.cluster_ca_certificate
}
`, version, time.Now().UnixNano())

	branch := map[string]func() (*schema.Provider, error){
		"minikube": func() (*schema.Provider, error) { return Provider(), nil },
	}
	resource.Test(t, resource.TestCase{
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: config,
				ExternalProviders: map[string]resource.ExternalProvider{
					"minikube": {Source: "scott-the-programmer/minikube"},
				},
				Check: resource.TestCheckResourceAttrSet("minikube_cluster.this", "host"),
			},
			{
				Config:            config,
				ProviderFactories: branch,
				PlanOnly:          true,
				// ExpectNonEmptyPlan defaults to false: fail before applying any diff.
			},
			{
				Config:            config,
				ProviderFactories: branch,
				// Reapply and let the harness verify the post-apply plan is empty.
			},
		},
	})
}
