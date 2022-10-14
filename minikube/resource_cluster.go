package minikube

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"terraform-provider-minikube/m/v2/minikube/service"
	"terraform-provider-minikube/m/v2/minikube/state_utils"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
)

var (
	defaultIso = service.GetMinikubeIso()
)

func ResourceCluster() *schema.Resource {
	return &schema.Resource{
		Description:   "Used to create a minikube cluster on the current host",
		CreateContext: resourceClusterCreate,
		ReadContext:   resourceClusterRead,
		DeleteContext: resourceClusterDelete,
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
	config := client.GetClusterConfig()

	addons := make([]string, 0, len(config.Addons))
	for k, v := range config.Addons {
		if v {
			addons = append(addons, k)
		}
	}
	sort.Strings(addons) //to ensure consistency with TF state

	if err != nil {
		return diag.FromErr(err)
	}

	stringPorts := config.ExposedPorts
	ports := make([]int, len(stringPorts))
	for i, sp := range stringPorts {
		p, _ := strconv.Atoi(sp)
		ports[i] = p
	}

	setClusterState(d, config, ports, addons)

	return diags
}

func setClusterState(d *schema.ResourceData, config *config.ClusterConfig, ports []int, addons []string) {

	d.Set("addons", addons)
	d.Set("apiserver_ips", state_utils.SliceOrNil(config.KubernetesConfig.APIServerIPs))
	d.Set("apiserver_name", config.KubernetesConfig.APIServerName)
	d.Set("apiserver_names", state_utils.SliceOrNil(config.KubernetesConfig.APIServerNames))
	d.Set("apiserver_port", config.KubernetesConfig.NodePort)
	d.Set("base_image", config.KicBaseImage)
	d.Set("cert_expiration", config.CertExpiration.Hours())
	d.Set("cni", config.KubernetesConfig.CNI)
	d.Set("container_runtime", config.KubernetesConfig.ContainerRuntime)
	d.Set("cpus", config.CPUs)
	d.Set("cri_socket", config.KubernetesConfig.CRISocket)
	d.Set("disable_driver_mounts", config.DisableDriverMounts)
	d.Set("disk_size", config.DiskSize)
	d.Set("dns_domain", config.KubernetesConfig.DNSDomain)
	d.Set("dns_proxy", config.DNSProxy)
	d.Set("driver", config.Driver)
	d.Set("embed_certs", config.EmbedCerts)
	d.Set("extra_disks", config.ExtraDisks)
	d.Set("feature_gates", config.KubernetesConfig.FeatureGates)
	d.Set("host_dns_resolver", config.HostDNSResolver)
	d.Set("host_only_cidr", config.HostOnlyCIDR)
	d.Set("host_only_nic_type", config.HostOnlyNicType)
	d.Set("hyperkit_vpnkit_sock", config.HyperkitVpnKitSock)
	d.Set("hyperkit_vsock_ports", state_utils.SliceOrNil(config.HyperkitVSockPorts))
	d.Set("hyperv_external_adapter", config.HypervExternalAdapter)
	d.Set("hyperv_use_external_switch", config.HypervUseExternalSwitch)
	d.Set("hyperv_virtual_switch", config.HypervVirtualSwitch)
	d.Set("image_repository", config.KubernetesConfig.ImageRepository)
	d.Set("insecure_registry", config.InsecureRegistry)
	d.Set("iso_url", []string{config.MinikubeISO})
	d.Set("keep_context", config.KeepContext)
	d.Set("kvm_gpu", config.KVMGPU)
	d.Set("kvm_hidden", config.KVMHidden)
	d.Set("kvm_network", config.KVMNetwork)
	d.Set("kvm_numa_count", config.KVMNUMACount)
	d.Set("kvm_qemu_uri", config.KVMQemuURI)
	d.Set("listen_address", config.ListenAddress)
	d.Set("memory", config.Memory)
	d.Set("mount", config.Mount)
	d.Set("mount_string", config.MountString)
	d.Set("namespace", config.KubernetesConfig.Namespace)
	d.Set("nat_nic_type", config.NatNicType)
	d.Set("network", config.Network)
	d.Set("network_plugin", config.KubernetesConfig.NetworkPlugin)
	d.Set("nfs_share", state_utils.SliceOrNil(config.NFSShare))
	d.Set("nfs_shares_root", config.NFSSharesRoot)
	d.Set("no_vtx_check", config.NoVTXCheck)
	d.Set("nodes", len(config.Nodes))
	d.Set("ports", state_utils.SliceOrNil(ports))
	d.Set("registry_mirror", state_utils.SliceOrNil(config.RegistryMirror))
	d.Set("service_cluster_ip_range", config.KubernetesConfig.ServiceCIDR)
	d.Set("ssh_ip_address", config.SSHIPAddress)
	d.Set("ssh_key", config.SSHKey)
	d.Set("ssh_port", config.SSHPort)
	d.Set("ssh_user", config.SSHUser)
	d.Set("uuid", config.UUID)
	d.Set("vm_driver", config.VMDriver)
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

func initialiseMinikubeClient(d *schema.ResourceData, m interface{}) (service.ClusterClient, error) {

	clusterClientFactory := m.(func() (service.ClusterClient, error))
	clusterClient, err := clusterClientFactory()
	if err != nil {
		return nil, err
	}

	addonStrings := getAddons(d)

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

	k8sVersion := clusterClient.GetK8sVersion()
	kubernetesConfig := config.KubernetesConfig{
		KubernetesVersion: k8sVersion,
		ClusterName:       d.Get("cluster_name").(string),
		Namespace:         d.Get("namespace").(string),
		APIServerName:     d.Get("apiserver_name").(string),
		APIServerNames:    []string{d.Get("apiserver_name").(string)},
		DNSDomain:         d.Get("dns_domain").(string),
		FeatureGates:      d.Get("feature_gates").(string),
		ContainerRuntime:  d.Get("container_runtime").(string),
		CRISocket:         d.Get("cri_socket").(string),
		NetworkPlugin:     d.Get("network_plugin").(string),
		ServiceCIDR:       d.Get("service_cluster_ip_range").(string),
		ImageRepository:   "",
		// ExtraOptions:           d.Get("extra_config").(string),
		ShouldLoadCachedImages: d.Get("cache_images").(bool),
		CNI:                    d.Get("cni").(string),
		NodePort:               d.Get("apiserver_port").(int),
	}

	n := config.Node{
		Name:              "",
		Port:              8443,
		KubernetesVersion: k8sVersion,
		ContainerRuntime:  d.Get("container_runtime").(string),
		ControlPlane:      true,
		Worker:            true,
	}

	cc := config.ClusterConfig{
		Name:                    d.Get("cluster_name").(string),
		KeepContext:             d.Get("keep_context").(bool),
		EmbedCerts:              d.Get("embed_certs").(bool),
		MinikubeISO:             state_utils.ReadSliceState(defaultIsos)[0],
		KicBaseImage:            d.Get("base_image").(string),
		Network:                 d.Get("network").(string),
		Memory:                  d.Get("memory").(int),
		CPUs:                    d.Get("cpus").(int),
		DiskSize:                d.Get("disk_size").(int),
		Driver:                  d.Get("driver").(string),
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
		StartHostTimeout:        time.Duration(d.Get("wait_timeout").(int)) * time.Second,
		ExposedPorts:            state_utils.ReadSliceState(ports),
		SSHIPAddress:            d.Get("ssh_ip_address").(string),
		SSHUser:                 d.Get("ssh_user").(string),
		SSHKey:                  d.Get("ssh_key").(string),
		SSHPort:                 d.Get("ssh_port").(int),
		ExtraDisks:              d.Get("extra_disks").(int),
		CertExpiration:          time.Duration(d.Get("cert_expiration").(int)) * time.Hour,
		Mount:                   d.Get("hyperv_use_external_switch").(bool),
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
		KubernetesConfig:   kubernetesConfig,
		MultiNodeRequested: false,
	}

	clusterClient.SetConfig(service.MinikubeClientArgs{
		ClusterConfig: cc, ClusterName: d.Get("cluster_name").(string),
		Addons:          addonStrings,
		IsoUrls:         state_utils.ReadSliceState(defaultIsos),
		DeleteOnFailure: d.Get("delete_on_failure").(bool),
		Nodes:           d.Get("nodes").(int),
	})

	clusterClient.SetDependencies(service.MinikubeClientDeps{
		Node:       service.NewMinikubeCluster(),
		Downloader: service.NewMinikubeDownloader(),
	})

	return clusterClient, nil
}

func getAddons(d *schema.ResourceData) []string {
	addons, ok := d.GetOk("addons")
	if !ok {
		addons = []interface{}{}
	}

	addonStrings := make([]string, len(addons.([]interface{})))
	userDefinedStorageClass := false
	for i, v := range addons.([]interface{}) {
		if v == "default-storageclass" {
			userDefinedStorageClass = true
		}
		addonStrings[i] = v.(string)
	}

	if !userDefinedStorageClass {
		addonStrings = append(addonStrings, "default-storageclass")
	}

	sort.Strings(addonStrings) //to ensure consistency with TF state

	return addonStrings
}
