package minikube

import (
	"context"
	"terraform-provider-minikube/m/v2/minikube/service"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestProvider(t *testing.T) {
	err := Provider().InternalValidate()
	assert.NoError(t, err)
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func TestProvider_bootstrap(t *testing.T) {
	provider := Provider()

	sch := map[string]*schema.Schema{
		"kubernetes_version": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The Kubernetes version that the minikube VM will use. Defaults to 'stable'.",
			Default:     "v99.99.99",
		},
	}

	rawC := make(map[string]interface{})

	data := schema.TestResourceDataRaw(t, sch, rawC)

	m, _ := provider.ConfigureContextFunc(context.TODO(), data)

	clusterClientFactory := m.(func() (service.ClusterClient, error))
	_, err := clusterClientFactory()

	assert.NoError(t, err)
}
