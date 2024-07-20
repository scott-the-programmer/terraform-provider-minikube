package minikube

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/state_utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	pkgutil "k8s.io/minikube/pkg/util"
)

var (
	defaultIso = lib.GetMinikubeIso()
)

func ResourceCluster() *schema.Resource {
	return &schema.Resource{
		Description:   "Used to create a minikube cluster on the current host",
		CreateContext: resourceClusterCreate,
		ReadContext:   resourceClusterRead,
		DeleteContext: resourceClusterDelete,
		UpdateContext: resourceClusterUpdate,
		Schema:        GetClusterSchema(),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceClusterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client, err := initialiseMinikubeClient(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	kc, err := client.Start()
	if err != nil {
		return diag.FromErr(err)
	}

	key, certificate, ca, address, err := getClusterOutputs(kc)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(d.Get("cluster_name").(string))
	d.Set("client_key", key)
	d.Set("client_certificate", certificate)
	d.Set("cluster_ca_certificate", ca)
	d.Set("host", address)
	d.Set("cluster_name", kc.ClusterName)

	diags = resourceClusterRead(ctx, d, m)

	return diags
}

func resourceClusterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client, err := initialiseMinikubeClient(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("addons") {
		config := client.GetConfig()
		oldAddons, newAddons := d.GetChange("addons")
		oldAddonStrings := getAddons(oldAddons.(*schema.Set))
		newAddonStrings := getAddons(newAddons.(*schema.Set))

		client.SetConfig(lib.MinikubeClientConfig{
			ClusterConfig:   config.ClusterConfig,
			IsoUrls:         config.IsoUrls,
			ClusterName:     config.ClusterName,
			Addons:          oldAddonStrings,
			DeleteOnFailure: config.DeleteOnFailure,
			Nodes:           config.Nodes,
		})

		err = client.ApplyAddons(newAddonStrings)
		if err != nil {
			return diag.FromErr(err)
		}

		sort.Strings(newAddonStrings) //to ensure consistency with TF state

		d.Set("addons", newAddonStrings)
	}

	return diags
}

func resourceClusterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client, err := initialiseMinikubeClient(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	err = client.Delete()
	if err != nil {
		fmt.Printf("Failed to delete cluster - you might want to consider running `minikube delete -p %s`", d.Get("cluster_name").(string))
	}

	d.SetId("")

	return diags
}

func resourceClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client, err := initialiseMinikubeClient(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	cc := client.GetClusterConfig()
	tfc := client.GetConfig()
	addons := client.GetAddons()
	sort.Strings(addons) //to ensure consistency with TF state

	stringPorts := cc.ExposedPorts
	ports := make([]int, len(stringPorts))
	for i, sp := range stringPorts {
		p, _ := strconv.Atoi(sp)
		ports[i] = p
	}

	setClusterState(d, cc, tfc, ports, addons)

	return diags
}

func setClusterState(d *schema.ResourceData, cc *config.ClusterConfig, tfc lib.MinikubeClientConfig, ports []int, addons []string) {

	d.Set("addons", addons)
	d.Set("apiserver_ips", state_utils.SliceOrNil(cc.KubernetesConfig.APIServerIPs))
	d.Set("apiserver_name", cc.KubernetesConfig.APIServerName)
	d.Set("apiserver_names", state_utils.SliceOrNil(cc.KubernetesConfig.APIServerNames))
	d.Set("apiserver_port", cc.APIServerPort)
	d.Set("base_image", cc.KicBaseImage)
	d.Set("cert_expiration", cc.CertExpiration.Minutes())
	d.Set("cni", cc.KubernetesConfig.CNI)
	d.Set("container_runtime", cc.KubernetesConfig.ContainerRuntime)
	d.Set("cpus", cc.CPUs)
	d.Set("cri_socket", cc.KubernetesConfig.CRISocket)
	d.Set("disable_driver_mounts", cc.DisableDriverMounts)
	d.Set("disk_size", strconv.Itoa(cc.DiskSize)+"mb")
	d.Set("dns_domain", cc.KubernetesConfig.DNSDomain)
	d.Set("dns_proxy", cc.DNSProxy)
	d.Set("driver", cc.Driver)
	d.Set("embed_certs", cc.EmbedCerts)
	d.Set("extra_disks", cc.ExtraDisks)

	extra_config := []string{}
	for _, e := range cc.KubernetesConfig.ExtraOptions {
		extra_config = append(extra_config, fmt.Sprintf("%s.%s=%s", e.Component, e.Key, e.Value))
	}

	d.Set("extra_config", extra_config)
	d.Set("feature_gates", cc.KubernetesConfig.FeatureGates)
	d.Set("host_dns_resolver", cc.HostDNSResolver)
	d.Set("host_only_cidr", cc.HostOnlyCIDR)
	d.Set("host_only_nic_type", cc.HostOnlyNicType)
	d.Set("hyperkit_vpnkit_sock", cc.HyperkitVpnKitSock)
	d.Set("hyperkit_vsock_ports", state_utils.SliceOrNil(cc.HyperkitVSockPorts))
	d.Set("hyperv_external_adapter", cc.HypervExternalAdapter)
	d.Set("hyperv_use_external_switch", cc.HypervUseExternalSwitch)
	d.Set("hyperv_virtual_switch", cc.HypervVirtualSwitch)
	d.Set("image_repository", cc.KubernetesConfig.ImageRepository)
	d.Set("insecure_registry", cc.InsecureRegistry)
	d.Set("iso_url", []string{cc.MinikubeISO})
	d.Set("keep_context", cc.KeepContext)
	d.Set("kvm_gpu", cc.KVMGPU)
	d.Set("kvm_hidden", cc.KVMHidden)
	d.Set("kvm_network", cc.KVMNetwork)
	d.Set("kvm_numa_count", cc.KVMNUMACount)
	d.Set("kvm_qemu_uri", cc.KVMQemuURI)
	d.Set("listen_address", cc.ListenAddress)
	d.Set("memory", strconv.Itoa(cc.Memory)+"mb")
	d.Set("mount", cc.Mount)
	d.Set("mount_string", cc.MountString)
	d.Set("namespace", cc.KubernetesConfig.Namespace)
	d.Set("nat_nic_type", cc.NatNicType)
	d.Set("network", cc.Network)
	d.Set("nfs_share", state_utils.SliceOrNil(cc.NFSShare))
	d.Set("nfs_shares_root", cc.NFSSharesRoot)
	d.Set("no_vtx_check", cc.NoVTXCheck)
	d.Set("nodes", tfc.Nodes)
	d.Set("ports", state_utils.SliceOrNil(ports))
	d.Set("registry_mirror", state_utils.SliceOrNil(cc.RegistryMirror))
	d.Set("service_cluster_ip_range", cc.KubernetesConfig.ServiceCIDR)
	d.Set("ssh_ip_address", cc.SSHIPAddress)
	d.Set("ssh_key", cc.SSHKey)
	d.Set("ssh_port", cc.SSHPort)
	d.Set("ssh_user", cc.SSHUser)
	d.Set("uuid", cc.UUID)
	d.Set("driver", cc.Driver)
}

// getClusterOutputs return the cluster key, certificate and certificate authority from the provided kubeconfig
func getClusterOutputs(kc *kubeconfig.Settings) (string, string, string, string, error) {
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

	if err != nil {
		return "", "", "", "", err
	}

	return key, certificate, ca, kc.ClusterServerAddress, nil
}

func initialiseMinikubeClient(d *schema.ResourceData, m interface{}) (lib.ClusterClient, error) {

	clusterClientFactory := m.(func() (lib.ClusterClient, error))
	clusterClient, err := clusterClientFactory()
	if err != nil {
		return nil, err
	}

	driver := d.Get("driver").(string)
	containerRuntime := d.Get("container_runtime").(string)

	addons, ok := d.GetOk("addons")
	if !ok {
		addons = &schema.Set{}
	}

	addonStrings := getAddons(addons.(*schema.Set))

	defaultIsos, ok := d.GetOk("iso_url")
	if !ok {
		defaultIsos = []string{defaultIso}
	}

	hyperKitSockPorts, ok := d.GetOk("hyperkit_vsock_ports")
	if !ok {
		hyperKitSockPorts = []string{}
	}

	nfsShare, ok := d.GetOk("nfs_share")
	if !ok {
		nfsShare = []string{}
	}

	ports, ok := d.GetOk("ports")
	if !ok {
		ports = []string{}
	}

	memoryStr := d.Get("memory").(string)
	memoryMb, err := pkgutil.CalculateSizeInMB(memoryStr)
	if err != nil {
		return nil, err
	}

	diskStr := d.Get("disk_size").(string)
	diskMb, err := pkgutil.CalculateSizeInMB(diskStr)
	if err != nil {
		return nil, err
	}

	apiserverNames := []string{}
	if d.Get("apiserver_names").(*schema.Set).Len() > 0 {
		apiserverNames = state_utils.ReadSliceState(d.Get("apiserver_names"))
	}

	networkPlugin := d.Get("network_plugin").(string) // This is a deprecated parameter in Minikube, however,
	// it is still used internally, so we need to set it to a default value if it is not set. We should expect
	// this to be a blank string usually, which should default to cni
	// Upstream : https://github.com/kubernetes/minikube/blob/37eeaddf7ad63a7f690129247650e8dd4ff3d56a/cmd/minikube/cmd/start_flags.go#L506-L514
	if networkPlugin == "" {
		networkPlugin = "cni"
	}

	ecSlice := []string{}
	if d.Get("extra_config") != nil && d.Get("extra_config").(*schema.Set).Len() > 0 {
		ecSlice = state_utils.ReadSliceState(d.Get("extra_config"))
	}

	var extraConfigs config.ExtraOptionSlice
	for _, e := range ecSlice {
		if err := extraConfigs.Set(e); err != nil {
			return nil, fmt.Errorf("invalid extra option: %s: %v", e, err)
		}
	}

	k8sVersion := clusterClient.GetK8sVersion()
	kubernetesConfig := config.KubernetesConfig{
		KubernetesVersion:      k8sVersion,
		ClusterName:            d.Get("cluster_name").(string),
		Namespace:              d.Get("namespace").(string),
		APIServerName:          d.Get("apiserver_name").(string),
		APIServerNames:         apiserverNames,
		DNSDomain:              d.Get("dns_domain").(string),
		FeatureGates:           d.Get("feature_gates").(string),
		ContainerRuntime:       containerRuntime,
		CRISocket:              d.Get("cri_socket").(string),
		NetworkPlugin:          networkPlugin,
		ServiceCIDR:            d.Get("service_cluster_ip_range").(string),
		ImageRepository:        "",
		ExtraOptions:           extraConfigs,
		ShouldLoadCachedImages: d.Get("cache_images").(bool),
		CNI:                    d.Get("cni").(string),
	}

	n := config.Node{
		Name:              "",
		Port:              8443,
		KubernetesVersion: k8sVersion,
		ContainerRuntime:  containerRuntime,
		ControlPlane:      true,
		Worker:            true,
	}

	addonConfig := make(map[string]bool)
	for _, addon := range addonStrings {
		addonConfig[addon] = true
	}

	nodes := d.Get("nodes").(int)
	multiNode := false

	if nodes > 1 {
		multiNode = true
	}

	if nodes == 0 {
		return nil, errors.New("at least one node is required")
	}

	ha := d.Get("ha").(bool)

	if ha && nodes < 3 {
		return nil, errors.New("at least 3 nodes is required for high availability")
	}

	cc := config.ClusterConfig{
		Addons:                  addonConfig,
		APIServerPort:           d.Get("apiserver_port").(int),
		Name:                    d.Get("cluster_name").(string),
		KeepContext:             d.Get("keep_context").(bool),
		EmbedCerts:              d.Get("embed_certs").(bool),
		MinikubeISO:             state_utils.ReadSliceState(defaultIsos)[0],
		KicBaseImage:            d.Get("base_image").(string),
		Network:                 d.Get("network").(string),
		Memory:                  memoryMb,
		CPUs:                    d.Get("cpus").(int),
		DiskSize:                diskMb,
		Driver:                  driver,
		ListenAddress:           d.Get("listen_address").(string),
		HyperkitVpnKitSock:      d.Get("hyperkit_vpnkit_sock").(string),
		HyperkitVSockPorts:      state_utils.ReadSliceState(hyperKitSockPorts),
		NFSShare:                state_utils.ReadSliceState(nfsShare),
		NFSSharesRoot:           d.Get("nfs_shares_root").(string),
		DockerEnv:               config.DockerEnv,
		DockerOpt:               config.DockerOpt,
		HostOnlyCIDR:            d.Get("host_only_cidr").(string),
		HypervVirtualSwitch:     d.Get("hyperv_virtual_switch").(string),
		HypervUseExternalSwitch: d.Get("hyperv_use_external_switch").(bool),
		HypervExternalAdapter:   d.Get("hyperv_external_adapter").(string),
		KVMNetwork:              d.Get("kvm_network").(string),
		KVMQemuURI:              d.Get("kvm_qemu_uri").(string),
		KVMGPU:                  d.Get("kvm_gpu").(bool),
		KVMHidden:               d.Get("kvm_hidden").(bool),
		KVMNUMACount:            d.Get("kvm_numa_count").(int),
		DisableDriverMounts:     d.Get("disable_driver_mounts").(bool),
		UUID:                    d.Get("uuid").(string),
		NoVTXCheck:              d.Get("no_vtx_check").(bool),
		DNSProxy:                d.Get("dns_proxy").(bool),
		HostDNSResolver:         d.Get("host_dns_resolver").(bool),
		HostOnlyNicType:         d.Get("host_only_nic_type").(string),
		NatNicType:              d.Get("host_only_nic_type").(string),
		StartHostTimeout:        time.Duration(d.Get("wait_timeout").(int)) * time.Minute,
		ExposedPorts:            state_utils.ReadSliceState(ports),
		SSHIPAddress:            d.Get("ssh_ip_address").(string),
		SSHUser:                 d.Get("ssh_user").(string),
		SSHKey:                  d.Get("ssh_key").(string),
		SSHPort:                 d.Get("ssh_port").(int),
		ExtraDisks:              d.Get("extra_disks").(int),
		CertExpiration:          time.Duration(d.Get("cert_expiration").(int)) * time.Minute,
		Mount:                   d.Get("mount").(bool),
		MountString:             d.Get("mount_string").(string),
		Mount9PVersion:          "9p2000.L",
		MountGID:                "docker",
		MountIP:                 "",
		MountMSize:              262144,
		MountOptions:            []string{},
		MountPort:               0,
		MountType:               "9p",
		MountUID:                "docker",
		BinaryMirror:            "",
		DisableOptimizations:    d.Get("hyperv_use_external_switch").(bool),
		Nodes: []config.Node{
			n,
		},
		KubernetesConfig:      kubernetesConfig,
		MultiNodeRequested:    multiNode,
		StaticIP:              d.Get("static_ip").(string),
		GPUs:                  d.Get("gpus").(string),
		SocketVMnetPath:       d.Get("socket_vmnet_path").(string),
		SocketVMnetClientPath: d.Get("socket_vmnet_client_path").(string),
	}

	clusterClient.SetConfig(lib.MinikubeClientConfig{
		ClusterConfig: &cc, ClusterName: d.Get("cluster_name").(string),
		Addons:          addonStrings,
		IsoUrls:         state_utils.ReadSliceState(defaultIsos),
		DeleteOnFailure: d.Get("delete_on_failure").(bool),
		Nodes:           nodes,
		HA:              ha,
		NativeSsh:       d.Get("native_ssh").(bool),
	})

	clusterClient.SetDependencies(lib.MinikubeClientDeps{
		Node:       lib.NewMinikubeCluster(),
		Downloader: lib.NewMinikubeDownloader(),
	})

	return clusterClient, nil
}

func getAddons(addons *schema.Set) []string {
	addonStrings := make([]string, addons.Len())
	addonObjects := addons.List()
	for i, v := range addonObjects {
		addonStrings[i] = v.(string)
	}

	sort.Strings(addonStrings) //to ensure consistency with TF state

	return addonStrings
}
