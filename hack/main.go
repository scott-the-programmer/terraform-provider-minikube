package main

import (
	"os"
	"terraform-provider-minikube/m/v2/minikube"
	"terraform-provider-minikube/m/v2/minikube/service"
	"time"

	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/mustload"
)

func main() {

	driver := os.Args[1]

	schema := minikube.ResourceCluster().Schema

	k8sVersion := "v1.23.3"
	kubernetesConfig := config.KubernetesConfig{
		KubernetesVersion: k8sVersion,
		ClusterName:       "terraform-provider-minikube-acc",
		Namespace:         schema["namespace"].Default.(string),
		APIServerName:     schema["apiserver_name"].Default.(string),
		APIServerNames:    []string{schema["apiserver_name"].Default.(string)},
		DNSDomain:         schema["dns_domain"].Default.(string),
		FeatureGates:      schema["feature_gates"].Default.(string),
		ContainerRuntime:  schema["container_runtime"].Default.(string),
		CRISocket:         schema["cri_socket"].Default.(string),
		NetworkPlugin:     schema["network_plugin"].Default.(string),
		ServiceCIDR:       schema["service_cluster_ip_range"].Default.(string),
		ImageRepository:   "",
		// ExtraOptions:           schema["extra_config"].Default.(string),
		ShouldLoadCachedImages: schema["cache_images"].Default.(bool),
		CNI:                    schema["cni"].Default.(string),
		NodePort:               schema["apiserver_port"].Default.(int),
	}

	n := config.Node{
		Name:              "",
		Port:              8443,
		KubernetesVersion: k8sVersion,
		ContainerRuntime:  "docker",
		ControlPlane:      true,
		Worker:            true,
	}

	cc := config.ClusterConfig{
		Name:                    "terraform-provider-minikube-acc",
		KeepContext:             schema["keep_context"].Default.(bool),
		EmbedCerts:              schema["embed_certs"].Default.(bool),
		MinikubeISO:             "https://github.com/kubernetes/minikube/releases/download/v1.25.2/minikube-v1.25.2.iso",
		KicBaseImage:            schema["base_image"].Default.(string),
		Network:                 schema["network"].Default.(string),
		Memory:                  6000,
		CPUs:                    2,
		DiskSize:                10000,
		Driver:                  driver,
		ListenAddress:           schema["listen_address"].Default.(string),
		HyperkitVpnKitSock:      schema["hyperkit_vpnkit_sock"].Default.(string),
		HyperkitVSockPorts:      []string{},
		NFSShare:                []string{},
		NFSSharesRoot:           schema["nfs_shares_root"].Default.(string),
		DockerEnv:               config.DockerEnv,
		DockerOpt:               config.DockerOpt,
		HostOnlyCIDR:            schema["host_only_cidr"].Default.(string),
		HypervVirtualSwitch:     schema["hyperv_virtual_switch"].Default.(string),
		HypervUseExternalSwitch: schema["hyperv_use_external_switch"].Default.(bool),
		HypervExternalAdapter:   schema["hyperv_external_adapter"].Default.(string),
		KVMNetwork:              schema["kvm_network"].Default.(string),
		KVMQemuURI:              schema["kvm_qemu_uri"].Default.(string),
		KVMGPU:                  schema["kvm_gpu"].Default.(bool),
		KVMHidden:               schema["kvm_hidden"].Default.(bool),
		KVMNUMACount:            schema["kvm_numa_count"].Default.(int),
		DisableDriverMounts:     schema["disable_driver_mounts"].Default.(bool),
		UUID:                    schema["uuid"].Default.(string),
		NoVTXCheck:              schema["no_vtx_check"].Default.(bool),
		DNSProxy:                schema["dns_proxy"].Default.(bool),
		HostDNSResolver:         schema["host_dns_resolver"].Default.(bool),
		HostOnlyNicType:         schema["host_only_nic_type"].Default.(string),
		NatNicType:              schema["host_only_nic_type"].Default.(string),
		StartHostTimeout:        time.Duration(600 * time.Second),
		ExposedPorts:            []string{},
		SSHIPAddress:            schema["ssh_ip_address"].Default.(string),
		SSHUser:                 schema["ssh_user"].Default.(string),
		SSHKey:                  schema["ssh_key"].Default.(string),
		SSHPort:                 schema["ssh_port"].Default.(int),
		ExtraDisks:              schema["extra_disks"].Default.(int),
		CertExpiration:          time.Duration(600 * 600 * time.Second),
		Mount:                   schema["hyperv_use_external_switch"].Default.(bool),
		MountString:             schema["mount_string"].Default.(string),
		Mount9PVersion:          "9p2000.L",
		MountGID:                "docker",
		MountIP:                 "",
		MountMSize:              262144,
		MountOptions:            []string{},
		MountPort:               0,
		MountType:               "9p",
		MountUID:                "docker",
		BinaryMirror:            "",
		DisableOptimizations:    schema["hyperv_use_external_switch"].Default.(bool),
		Nodes: []config.Node{
			n,
		},
		// DisableMetrics:          schema["hyperv_use_external_switch"].Default.(bool),
		KubernetesConfig:   kubernetesConfig,
		MultiNodeRequested: false,
	}

	minikubeClient := service.NewMinikubeClient(
		service.MinikubeClientArgs{
			ClusterConfig:   cc,
			ClusterName:     "terraform-provider-minikube-acc",
			Addons:          []string{},
			IsoUrls:         []string{"https://github.com/kubernetes/minikube/releases/download/v1.25.2/minikube-v1.25.2.iso"},
			DeleteOnFailure: true},
		service.MinikubeClientDeps{
			Node:       service.NewMinikubeNode(),
			Downloader: service.NewMinikubeDownloader(),
		})

	kc, err := minikubeClient.Start()

	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	co := mustload.Running(kc.ClusterName)
	kubeconfig.UpdateEndpoint(kc.ClusterName, co.CP.Hostname, co.CP.Port, kubeconfig.PathFromEnv(), kubeconfig.NewExtension())
	kubeconfig.SetCurrentContext(kc.ClusterName, kubeconfig.PathFromEnv())

	minikubeClient.Delete()
}
