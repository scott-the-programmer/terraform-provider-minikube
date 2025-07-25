package minikube

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/state_utils"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	pkgutil "k8s.io/minikube/pkg/util"
)

var (
	_ resource.Resource                = &ClusterResource{}
	_ resource.ResourceWithImportState = &ClusterResource{}
	
	defaultIso = lib.GetMinikubeIso()
)

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

// ClusterResource defines the resource implementation.
type ClusterResource struct {
	clientFactory func() (lib.ClusterClient, error)
}

// ClusterResourceModel describes the resource data model.
type ClusterResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	ClusterName               types.String `tfsdk:"cluster_name"`
	Driver                    types.String `tfsdk:"driver"`
	ContainerRuntime          types.String `tfsdk:"container_runtime"`
	Memory                    types.String `tfsdk:"memory"`
	CPUs                      types.String `tfsdk:"cpus"`
	DiskSize                  types.String `tfsdk:"disk_size"`
	Nodes                     types.Int64  `tfsdk:"nodes"`
	CacheImages               types.Bool   `tfsdk:"cache_images"`
	DeleteOnFailure           types.Bool   `tfsdk:"delete_on_failure"`
	Namespace                 types.String `tfsdk:"namespace"`
	APIServerName             types.String `tfsdk:"apiserver_name"`
	DNSDomain                 types.String `tfsdk:"dns_domain"`
	FeatureGates              types.String `tfsdk:"feature_gates"`
	CRISocket                 types.String `tfsdk:"cri_socket"`
	NetworkPlugin             types.String `tfsdk:"network_plugin"`
	ServiceClusterIPRange     types.String `tfsdk:"service_cluster_ip_range"`
	CNI                       types.String `tfsdk:"cni"`
	
	// Sets/Lists
	Addons                    types.Set    `tfsdk:"addons"`
	APIServerIPs              types.Set    `tfsdk:"apiserver_ips"`
	APIServerNames            types.Set    `tfsdk:"apiserver_names"`
	IsoURL                    types.Set    `tfsdk:"iso_url"`
	HyperkitVsockPorts        types.Set    `tfsdk:"hyperkit_vsock_ports"`
	NFSShare                  types.Set    `tfsdk:"nfs_share"`
	Ports                     types.Set    `tfsdk:"ports"`
	ExtraConfig               types.Set    `tfsdk:"extra_config"`
	
	// Computed attributes
	ClientKey                 types.String `tfsdk:"client_key"`
	ClientCertificate         types.String `tfsdk:"client_certificate"`
	ClusterCACertificate      types.String `tfsdk:"cluster_ca_certificate"`
	Host                      types.String `tfsdk:"host"`
}

func (r *ClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Used to create a minikube cluster on the current host",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "The name of the minikube cluster",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("terraform-provider-minikube"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"driver": schema.StringAttribute{
				MarkdownDescription: "Driver is one of: virtualbox, vmwarefusion, kvm2, vmware, none, docker, podman, ssh",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("docker"),
			},
			"container_runtime": schema.StringAttribute{
				MarkdownDescription: "The container runtime to be used",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("docker"),
			},
			"memory": schema.StringAttribute{
				MarkdownDescription: "Amount of RAM to allocate to the minikube VM (format: <number>[<unit>])",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("6000mb"),
			},
			"cpus": schema.StringAttribute{
				MarkdownDescription: "Number of CPUs allocated to the minikube VM",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("2"),
			},
			"disk_size": schema.StringAttribute{
				MarkdownDescription: "Disk size allocated to the minikube VM (format: <number>[<unit>])",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("20000mb"),
			},
			"nodes": schema.Int64Attribute{
				MarkdownDescription: "The total number of nodes to spin up. Defaults to 1",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
			},
			"cache_images": schema.BoolAttribute{
				MarkdownDescription: "If true, cache docker images for the current bootstrapper and load them into the machine",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"delete_on_failure": schema.BoolAttribute{
				MarkdownDescription: "If set, delete the current cluster if start fails and try again",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "The namespace to create",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
			},
			"apiserver_name": schema.StringAttribute{
				MarkdownDescription: "The authoritative apiserver hostname for apiserver certificates and connectivity",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("minikubeCA"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_domain": schema.StringAttribute{
				MarkdownDescription: "The cluster dns domain name used in the kubernetes cluster",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("cluster.local"),
			},
			"feature_gates": schema.StringAttribute{
				MarkdownDescription: "A set of key=value pairs that describe feature gates for alpha/experimental features",
				Optional:            true,
			},
			"cri_socket": schema.StringAttribute{
				MarkdownDescription: "The cri socket path to be used",
				Optional:            true,
			},
			"network_plugin": schema.StringAttribute{
				MarkdownDescription: "The name of the network plugin",
				Optional:            true,
			},
			"service_cluster_ip_range": schema.StringAttribute{
				MarkdownDescription: "The CIDR to be used for service cluster IPs",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("10.96.0.0/12"),
			},
			"cni": schema.StringAttribute{
				MarkdownDescription: "CNI plug-in to use",
				Optional:            true,
			},
			"addons": schema.SetAttribute{
				MarkdownDescription: "Enable addons. see `minikube addons list` for a list of valid addon names",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"apiserver_ips": schema.SetAttribute{
				MarkdownDescription: "A set of apiserver IP Addresses which are used in the generated certificate for kubernetes",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				PlanModifiers: []planmodifier.Set{},
			},
			"apiserver_names": schema.SetAttribute{
				MarkdownDescription: "A set of apiserver names which are used in the generated certificate for kubernetes",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				PlanModifiers: []planmodifier.Set{},
			},
			"iso_url": schema.SetAttribute{
				MarkdownDescription: "Location of the minikube iso",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"hyperkit_vsock_ports": schema.SetAttribute{
				MarkdownDescription: "List of guest VSock ports that should be exposed as sockets on the host",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"nfs_share": schema.SetAttribute{
				MarkdownDescription: "Local folders to share with Guest via NFS mounts",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"ports": schema.SetAttribute{
				MarkdownDescription: "List of ports that should be exposed",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"extra_config": schema.SetAttribute{
				MarkdownDescription: "A set of key=value pairs that describe configuration that may be passed to different components",
				Optional:            true,
				ElementType:         types.StringType,
			},
			
			// Computed attributes
			"client_key": schema.StringAttribute{
				MarkdownDescription: "client key for cluster",
				Computed:            true,
				Sensitive:           true,
			},
			"client_certificate": schema.StringAttribute{
				MarkdownDescription: "client certificate used in cluster",
				Computed:            true,
				Sensitive:           true,
			},
			"cluster_ca_certificate": schema.StringAttribute{
				MarkdownDescription: "certificate authority for cluster",
				Computed:            true,
				Sensitive:           true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "the host name for the cluster",
				Computed:            true,
			},
		},
	}
}

func (r *ClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientFactory, ok := req.ProviderData.(func() (lib.ClusterClient, error))

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected func() (lib.ClusterClient, error), got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clientFactory = clientFactory
}

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create minikube client with the simplified type-safe access
	client, err := r.createMinikubeClient(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create minikube client, got error: %s", err))
		return
	}

	// Start the cluster
	kc, err := client.Start()
	if err != nil {
		resp.Diagnostics.AddError("Cluster Creation Error", fmt.Sprintf("Unable to start cluster, got error: %s", err))
		return
	}

	// Extract kubeconfig details
	key, certificate, ca, host, err := extractKubeconfig(kc, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Kubeconfig Error", fmt.Sprintf("Unable to extract kubeconfig, got error: %s", err))
		return
	}

	// Set computed values
	data.ID = types.StringValue(data.ClusterName.ValueString())
	data.ClientKey = types.StringValue(key)
	data.ClientCertificate = types.StringValue(certificate)
	data.ClusterCACertificate = types.StringValue(ca)
	data.Host = types.StringValue(host)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create client to check cluster status
	client, err := r.createMinikubeClient(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create minikube client, got error: %s", err))
		return
	}

	// Get current cluster configuration
	config := client.GetConfig()
	
	// Try to get kubeconfig through the cluster config
	clusterConfig := config.ClusterConfig
	if clusterConfig == nil {
		// If cluster doesn't exist, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// For read operation, we just check if cluster exists and maintain state
	// The actual kubeconfig details are computed during create/update

	// Update computed values (maintain existing state for read)
	// The kubeconfig details are already in state from create/update operations

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create minikube client
	client, err := r.createMinikubeClient(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create minikube client, got error: %s", err))
		return
	}

	// Apply addons (simplified from the original complex parsing)
	if !data.Addons.IsNull() {
		var addons []string
		resp.Diagnostics.Append(data.Addons.ElementsAs(ctx, &addons, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		err = client.ApplyAddons(addons)
		if err != nil {
			resp.Diagnostics.AddError("Addon Error", fmt.Sprintf("Unable to apply addons, got error: %s", err))
			return
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create minikube client
	client, err := r.createMinikubeClient(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create minikube client, got error: %s", err))
		return
	}

	// Delete the cluster
	err = client.Delete()
	if err != nil {
		resp.Diagnostics.AddError("Deletion Error", fmt.Sprintf("Unable to delete cluster, got error: %s", err))
		return
	}
}

func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import requires the path package
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), req.ID)...)
}

// createMinikubeClient creates a configured minikube client from the resource data
// This replaces the complex type parsing in the original initialiseMinikubeClient function
func (r *ClusterResource) createMinikubeClient(ctx context.Context, data *ClusterResourceModel) (lib.ClusterClient, error) {
	clusterClient, err := r.clientFactory()
	if err != nil {
		return nil, err
	}

	// Convert string values with type-safe access (no more manual type assertions!)
	driver := data.Driver.ValueString()
	containerRuntime := data.ContainerRuntime.ValueString()

	// Handle sets with type-safe extraction
	var addons []string
	if !data.Addons.IsNull() {
		data.Addons.ElementsAs(ctx, &addons, false)
	}

	var isoURLs []string
	if !data.IsoURL.IsNull() {
		data.IsoURL.ElementsAs(ctx, &isoURLs, false)
	} else {
		isoURLs = []string{defaultIso}
	}

	var hyperKitSockPorts []string
	if !data.HyperkitVsockPorts.IsNull() {
		data.HyperkitVsockPorts.ElementsAs(ctx, &hyperKitSockPorts, false)
	}

	var nfsShare []string
	if !data.NFSShare.IsNull() {
		data.NFSShare.ElementsAs(ctx, &nfsShare, false)
	}

	var ports []string
	if !data.Ports.IsNull() {
		data.Ports.ElementsAs(ctx, &ports, false)
	}

	var apiServerNames []string
	if !data.APIServerNames.IsNull() {
		data.APIServerNames.ElementsAs(ctx, &apiServerNames, false)
	}

	// Type-safe numeric conversions
	memoryMb, err := state_utils.GetMemory(data.Memory.ValueString())
	if err != nil {
		return nil, err
	}

	cpus, err := state_utils.GetCPUs(data.CPUs.ValueString())
	if err != nil {
		return nil, err
	}

	diskMb, err := pkgutil.CalculateSizeInMB(data.DiskSize.ValueString())
	if err != nil {
		return nil, err
	}

	// Handle network plugin default
	networkPlugin := data.NetworkPlugin.ValueString()
	if networkPlugin == "" {
		networkPlugin = "cni"
	}

	// Handle extra config with type-safe access
	var extraConfigs config.ExtraOptionSlice
	if !data.ExtraConfig.IsNull() {
		var extraConfigSlice []string
		data.ExtraConfig.ElementsAs(ctx, &extraConfigSlice, false)
		
		for _, e := range extraConfigSlice {
			if err := extraConfigs.Set(e); err != nil {
				return nil, fmt.Errorf("invalid extra option: %s: %v", e, err)
			}
		}
	}

	// Variables needed for configuration
	nodes := int(data.Nodes.ValueInt64())
	multiNode := false
	if nodes > 1 {
		multiNode = true
	}

	var apiServerIPsStr []string
	if !data.APIServerIPs.IsNull() {
		data.APIServerIPs.ElementsAs(ctx, &apiServerIPsStr, false)
	}

	// Convert string IPs to net.IP
	var apiServerIPs []net.IP
	for _, ipStr := range apiServerIPsStr {
		if ip := net.ParseIP(ipStr); ip != nil {
			apiServerIPs = append(apiServerIPs, ip)
		}
	}

	addonConfig := make(map[string]bool)
	for _, addon := range addons {
		addonConfig[addon] = true
	}

	k8sVersion := clusterClient.GetK8sVersion()
	kubernetesConfig := config.KubernetesConfig{
		KubernetesVersion:      k8sVersion,
		ClusterName:            data.ClusterName.ValueString(),
		Namespace:              data.Namespace.ValueString(),
		APIServerName:          data.APIServerName.ValueString(),
		APIServerNames:         apiServerNames,
		APIServerIPs:           apiServerIPs,
		DNSDomain:              data.DNSDomain.ValueString(),
		FeatureGates:           data.FeatureGates.ValueString(),
		ContainerRuntime:       containerRuntime,
		CRISocket:              data.CRISocket.ValueString(),
		NetworkPlugin:          networkPlugin,
		ServiceCIDR:            data.ServiceClusterIPRange.ValueString(),
		ImageRepository:        "",
		ExtraOptions:           extraConfigs,
		ShouldLoadCachedImages: data.CacheImages.ValueBool(),
		CNI:                    data.CNI.ValueString(),
	}

	n := config.Node{
		Name:              "",
		Port:              8443,
		KubernetesVersion: k8sVersion,
		ContainerRuntime:  containerRuntime,
		ControlPlane:      true,
		Worker:            true,
	}

	cc := &config.ClusterConfig{
		Name:                    data.ClusterName.ValueString(),
		KeepContext:             false,
		EmbedCerts:              false,
		MinikubeISO:             isoURLs[0], // Use first ISO URL
		KicBaseImage:            "",
		Network:                 "",
		Memory:                  memoryMb,
		CPUs:                    cpus,
		DiskSize:                diskMb,
		Driver:                  driver,
		HyperkitVpnKitSock:      "",
		HyperkitVSockPorts:      hyperKitSockPorts,
		DockerEnv:               nil,
		DockerOpt:               nil,
		InsecureRegistry:        nil,
		RegistryMirror:          nil,
		HostOnlyCIDR:            "192.168.59.1/24",
		HypervVirtualSwitch:     "",
		KVMNetwork:              "default",
		KVMQemuURI:              "qemu:///system",
		KVMGPU:                  false,
		KVMHidden:               false,
		KVMNUMACount:            1,
		DisableDriverMounts:     false,
		NFSShare:                nfsShare,
		NFSSharesRoot:           "/nfsshares",
		UUID:                    "",
		NoVTXCheck:              false,
		DNSProxy:                false,
		HostDNSResolver:         true,
		HostOnlyNicType:         "virtio",
		NatNicType:              "virtio",
		StartHostTimeout:        6 * time.Minute,
		ExposedPorts:            ports,
		ListenAddress:           "",
		ExtraDisks:              0,
		CertExpiration:          time.Duration(0),
		Mount:                   true,
		MountString:             "",
		Mount9PVersion:          "9p2000.L",
		MountGID:                "docker",
		MountIP:                 "",
		MountMSize:              262144,
		MountOptions:            []string{},
		MountPort:               0,
		MountType:               "9p",
		MountUID:                "docker",
		BinaryMirror:            "",
		DisableOptimizations:    false,
		DisableMetrics:          false,
		Nodes:                   []config.Node{n},
		Addons:                  addonConfig,
		VerifyComponents:        map[string]bool{},
		ScheduledStop:           nil,
		KubernetesConfig:        kubernetesConfig,
		MultiNodeRequested:      multiNode,
	}

	clusterClient.SetConfig(lib.MinikubeClientConfig{
		ClusterConfig:   cc,
		ClusterName:     data.ClusterName.ValueString(),
		Addons:          addons,
		IsoUrls:         isoURLs,
		DeleteOnFailure: data.DeleteOnFailure.ValueBool(),
		Nodes:           nodes,
		HA:              multiNode,
		NativeSsh:       false,
	})

	clusterClient.SetDependencies(lib.MinikubeClientDeps{
		Node:       lib.NewMinikubeCluster(),
		Downloader: lib.NewMinikubeDownloader(),
	})

	return clusterClient, nil
}

// extractKubeconfig extracts kubeconfig details (simplified from original)
func extractKubeconfig(kc *kubeconfig.Settings, clusterName string) (string, string, string, string, error) {
	if kc == nil {
		return "", "", "", "", fmt.Errorf("kubeconfig is nil")
	}

	key, err := state_utils.ReadContents(kc.ClientKey)
	if err != nil {
		return "", "", "", "", err
	}

	certificate, err := state_utils.ReadContents(kc.ClientCertificate)
	if err != nil {
		return "", "", "", "", err
	}

	ca, err := state_utils.ReadContents(kc.CertificateAuthority)
	if err != nil {
		return "", "", "", "", err
	}

	return key, certificate, ca, kc.ClusterServerAddress, nil
}