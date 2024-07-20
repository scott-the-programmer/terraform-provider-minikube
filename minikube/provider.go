package minikube

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func Provider() *schema.Provider {
	return NewProvider(providerConfigure)
}

func NewProvider(providerConfigure schema.ConfigureContextFunc) *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"minikube_cluster": ResourceCluster(),
		},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
		Schema: map[string]*schema.Schema{
			"kubernetes_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Kubernetes version that the minikube VM will use. Defaults to 'v1.30.0'.",
				Default:     "v1.30.0",
			},
		},
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	mutex := &sync.Mutex{}
	k8sVersion := d.Get("kubernetes_version").(string)
	minikubeClientFactory := func() (lib.ClusterClient, error) {
		return &lib.MinikubeClient{
			TfCreationLock: mutex,
			K8sVersion:     k8sVersion}, nil
	}
	return minikubeClientFactory, diags
}
