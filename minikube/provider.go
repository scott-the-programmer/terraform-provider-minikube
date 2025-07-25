package minikube

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ provider.Provider = &MinikubeProvider{}

// MinikubeProvider defines the provider implementation.
type MinikubeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// MinikubeProviderModel describes the provider data model.
type MinikubeProviderModel struct {
	KubernetesVersion types.String `tfsdk:"kubernetes_version"`
}

func (p *MinikubeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "minikube"
	resp.Version = p.version
}

func (p *MinikubeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Minikube provider for Terraform",
		Attributes: map[string]schema.Attribute{
			"kubernetes_version": schema.StringAttribute{
				MarkdownDescription: "The Kubernetes version that the minikube VM will use. Defaults to 'v1.30.0'.",
				Optional:            true,
			},
		},
	}
}

func (p *MinikubeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MinikubeProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Set default kubernetes version if not provided
	k8sVersion := "v1.30.0"
	if !data.KubernetesVersion.IsNull() {
		k8sVersion = data.KubernetesVersion.ValueString()
	}

	// Create the client factory function
	mutex := &sync.Mutex{}
	minikubeClientFactory := func() (lib.ClusterClient, error) {
		return &lib.MinikubeClient{
			TfCreationLock: mutex,
			K8sVersion:     k8sVersion,
		}, nil
	}

	resp.DataSourceData = minikubeClientFactory
	resp.ResourceData = minikubeClientFactory
}

func (p *MinikubeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
	}
}

func (p *MinikubeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Define data sources here
	}
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MinikubeProvider{
			version: version,
		}
	}
}

// For backward compatibility with the old SDK name
func Provider() func() provider.Provider {
	return NewProvider("dev")
}