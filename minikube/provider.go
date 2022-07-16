package minikube

import (
	"context"
	"sync"
	"terraform-provider-minikube/m/v2/minikube/service"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func Provider() *schema.Provider {
	return NewProvider(providerConfigure)
}

func NewProvider(providerConfigure schema.ConfigureContextFunc) *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},
		ResourcesMap: map[string]*schema.Resource{
			"minikube_cluster": ResourceCluster(),
		},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	mutex := &sync.Mutex{}
	minikubeClientFactory := func() (service.ClusterClient, error) {
		return &service.MinikubeClient{TfCreationLock: mutex}, nil
	}
	return minikubeClientFactory, diags
}
