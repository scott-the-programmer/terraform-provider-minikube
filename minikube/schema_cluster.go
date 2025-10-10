//go:generate go run ../generate/main.go -target $GOFILE
// THIS FILE IS GENERATED DO NOT EDIT
package minikube

import (
	"runtime"
	"os"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/state_utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	clusterSchema = map[string]*schema.Schema{
		"cluster_name": {
			Type:					schema.TypeString,
			Optional:			true,
			ForceNew:			true,
			Description:	"The name of the minikube cluster",
			Default:			"terraform-provider-minikube",
		},

		"client_key": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "client key for cluster",
			Sensitive:   true,
		},

		"client_certificate": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "client certificate used in cluster",
			Sensitive:   true,
		},

		"cluster_ca_certificate": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "certificate authority for cluster",
			Sensitive:   true,
		},

		"host": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "the host name for the cluster",
		},

		"addons": {
			Type:					schema.TypeSet,
			Description:	"Enable addons. see `minikube addons list` for a list of valid addon names.",
			
			Optional:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"apiserver_ips": {
			Type:					schema.TypeSet,
			Description:	"A set of apiserver IP Addresses which are used in the generated certificate for kubernetes.  This can be used if you want to make the apiserver available from outside the machine",
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"apiserver_name": {
			Type:					schema.TypeString,
			Description:	"The authoritative apiserver hostname for apiserver certificates and connectivity. This can be used if you want to make the apiserver available from outside the machine",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"minikubeCA",
		},
	
		"apiserver_names": {
			Type:					schema.TypeSet,
			Description:	"A set of apiserver names which are used in the generated certificate for kubernetes.  This can be used if you want to make the apiserver available from outside the machine",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"apiserver_port": {
			Type:					schema.TypeInt,
			Description:	"The apiserver listening port",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	8443,
		},
	
		"auto_pause_interval": {
			Type:					schema.TypeInt,
			Description:	"Duration of inactivity before the minikube VM is paused (default 1m0s) (Configured in minutes)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	1,
		},
	
		"auto_update_drivers": {
			Type:					schema.TypeBool,
			Description:	"If set, automatically updates drivers to the latest version. Defaults to true.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"base_image": {
			Type:					schema.TypeString,
			Description:	"The base image to use for docker/podman drivers. Intended for local development.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"gcr.io/k8s-minikube/kicbase:v0.0.47@sha256:6ed579c9292b4370177b7ef3c42cc4b4a6dcd0735a1814916cbc22c8bf38412b",
		},
	
		"binary_mirror": {
			Type:					schema.TypeString,
			Description:	"Location to fetch kubectl, kubelet, & kubeadm binaries from.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"cache_images": {
			Type:					schema.TypeBool,
			Description:	"If true, cache docker images for the current bootstrapper and load them into the machine. Always false with --driver=none.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"cert_expiration": {
			Type:					schema.TypeInt,
			Description:	"Duration until minikube certificate expiration, defaults to three years (26280h). (Configured in minutes)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	1576800,
		},
	
		"cni": {
			Type:					schema.TypeString,
			Description:	"CNI plug-in to use. Valid options: auto, bridge, calico, cilium, flannel, kindnet, or path to a CNI manifest (default: auto)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"container_runtime": {
			Type:					schema.TypeString,
			Description:	"The container runtime to be used. Valid options: docker, cri-o, containerd (default: docker)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"docker",
		},
	
		"cpus": {
			Type:					schema.TypeString,
			Description:	"Number of CPUs allocated to Kubernetes. Use \"max\" to use the maximum number of CPUs. Use \"no-limit\" to not specify a limit (Docker/Podman only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"2",
			StateFunc:	state_utils.CPUConverter(),
			ValidateDiagFunc:	state_utils.CPUValidator(),
		},
	
		"cri_socket": {
			Type:					schema.TypeString,
			Description:	"The cri socket path to be used.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"delete_on_failure": {
			Type:					schema.TypeBool,
			Description:	"If set, delete the current cluster if start fails and try again. Defaults to false.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"disable_driver_mounts": {
			Type:					schema.TypeBool,
			Description:	"Disables the filesystem mounts provided by the hypervisors",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"disable_metrics": {
			Type:					schema.TypeBool,
			Description:	"If set, disables metrics reporting (CPU and memory usage), this can improve CPU usage. Defaults to false.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"disable_optimizations": {
			Type:					schema.TypeBool,
			Description:	"If set, disables optimizations that are set for local Kubernetes. Including decreasing CoreDNS replicas from 2 to 1. Defaults to false.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"disk_size": {
			Type:					schema.TypeString,
			Description:	"Disk size allocated to the minikube VM (format: <number>[<unit>(case-insensitive)], where unit = b, k, kb, m, mb, g or gb)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"20000mb",
			StateFunc:	state_utils.ResourceSizeConverter(),
			ValidateDiagFunc:	state_utils.ResourceSizeValidator(),
		},
	
		"dns_domain": {
			Type:					schema.TypeString,
			Description:	"The cluster dns domain name used in the Kubernetes cluster",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"cluster.local",
		},
	
		"dns_proxy": {
			Type:					schema.TypeBool,
			Description:	"Enable proxy for NAT DNS requests (virtualbox driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"docker_env": {
			Type:					schema.TypeSet,
			Description:	"Environment variables to pass to the Docker daemon. (format: key=value)",
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"docker_opt": {
			Type:					schema.TypeSet,
			Description:	"Specify arbitrary flags to pass to the Docker daemon. (format: key=value)",
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"download_only": {
			Type:					schema.TypeBool,
			Description:	"If true, only download and cache files for later use - don't install or start anything.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"driver": {
			Type:					schema.TypeString,
			Description:	"Driver is one of the following - Windows: (hyperv, docker, virtualbox, vmware, qemu2, ssh) - OSX: (virtualbox, parallels, vmwarefusion, hyperkit, vmware, qemu2, docker, podman, ssh) - Linux: (docker, kvm2, virtualbox, qemu2, none, podman, ssh)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"docker",
		},
	
		"dry_run": {
			Type:					schema.TypeBool,
			Description:	"dry-run mode. Validates configuration, but does not mutate system state",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"embed_certs": {
			Type:					schema.TypeBool,
			Description:	"if true, will embed the certs in kubeconfig.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"enable_default_cni": {
			Type:					schema.TypeBool,
			Description:	"DEPRECATED: Replaced by --cni=bridge",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"extra_config": {
			Type:					schema.TypeSet,
			Description:	"A set of key=value pairs that describe configuration that may be passed to different components. 		The key should be '.' separated, and the first part before the dot is the component to apply the configuration to. 		Valid components are: kubelet, kubeadm, apiserver, controller-manager, etcd, proxy, scheduler 		Valid kubeadm parameters: ignore-preflight-errors, dry-run, kubeconfig, kubeconfig-dir, node-name, cri-socket, experimental-upload-certs, certificate-key, rootfs, skip-phases, pod-network-cidr",
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"extra_disks": {
			Type:					schema.TypeInt,
			Description:	"Number of extra disks created and attached to the minikube VM (currently only implemented for hyperkit, kvm2, qemu2, and vfkit drivers)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	0,
		},
	
		"feature_gates": {
			Type:					schema.TypeString,
			Description:	"A set of key=value pairs that describe feature gates for alpha/experimental features.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"force": {
			Type:					schema.TypeBool,
			Description:	"Force minikube to perform possibly dangerous operations",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"force_systemd": {
			Type:					schema.TypeBool,
			Description:	"If set, force the container runtime to use systemd as cgroup manager. Defaults to false.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"gpus": {
			Type:					schema.TypeString,
			Description:	"Allow pods to use your GPUs. Options include: [all,nvidia,amd] (Docker driver with Docker container-runtime only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"ha": {
			Type:					schema.TypeBool,
			Description:	"Create Highly Available Multi-Control Plane Cluster with a minimum of three control-plane nodes that will also be marked for work.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"host_dns_resolver": {
			Type:					schema.TypeBool,
			Description:	"Enable host resolver for NAT DNS requests (virtualbox driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"host_only_cidr": {
			Type:					schema.TypeString,
			Description:	"The CIDR to be used for the minikube VM (virtualbox driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"192.168.59.1/24",
		},
	
		"host_only_nic_type": {
			Type:					schema.TypeString,
			Description:	"NIC Type used for host only network. One of Am79C970A, Am79C973, 82540EM, 82543GC, 82545EM, or virtio (virtualbox driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"virtio",
		},
	
		"hyperkit_vpnkit_sock": {
			Type:					schema.TypeString,
			Description:	"Location of the VPNKit socket used for networking. If empty, disables Hyperkit VPNKitSock, if 'auto' uses Docker for Mac VPNKit connection, otherwise uses the specified VSock (hyperkit driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"hyperkit_vsock_ports": {
			Type:					schema.TypeSet,
			Description:	"List of guest VSock ports that should be exposed as sockets on the host (hyperkit driver only)",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"hyperv_external_adapter": {
			Type:					schema.TypeString,
			Description:	"External Adapter on which external switch will be created if no external switch is found. (hyperv driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"hyperv_use_external_switch": {
			Type:					schema.TypeBool,
			Description:	"Whether to use external switch over Default Switch if virtual switch not explicitly specified. (hyperv driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"hyperv_virtual_switch": {
			Type:					schema.TypeString,
			Description:	"The hyperv virtual switch name. Defaults to first found. (hyperv driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"image_mirror_country": {
			Type:					schema.TypeString,
			Description:	"Country code of the image mirror to be used. Leave empty to use the global one. For Chinese mainland users, set it to cn.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"image_repository": {
			Type:					schema.TypeString,
			Description:	"Alternative image repository to pull docker images from. This can be used when you have limited access to gcr.io. Set it to \"auto\" to let minikube decide one for you. For Chinese mainland users, you may use local gcr.io mirrors such as registry.cn-hangzhou.aliyuncs.com/google_containers",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"insecure_registry": {
			Type:					schema.TypeSet,
			Description:	"Insecure Docker registries to pass to the Docker daemon.  The default service CIDR range will automatically be added.",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"install_addons": {
			Type:					schema.TypeBool,
			Description:	"If set, install addons. Defaults to true.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"interactive": {
			Type:					schema.TypeBool,
			Description:	"Allow user prompts for more information",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"iso_url": {
			Type:					schema.TypeSet,
			Description:	"Locations to fetch the minikube ISO from.",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"keep_context": {
			Type:					schema.TypeBool,
			Description:	"This will keep the existing kubectl context and will create a minikube context.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"kubernetes_version": {
			Type:					schema.TypeString,
			Description:	"The Kubernetes version that the minikube VM will use (ex: v1.2.3, 'stable' for v1.33.1, 'latest' for v1.33.1). Defaults to 'stable'.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"kvm_gpu": {
			Type:					schema.TypeBool,
			Description:	"Enable experimental NVIDIA GPU support in minikube",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"kvm_hidden": {
			Type:					schema.TypeBool,
			Description:	"Hide the hypervisor signature from the guest in minikube (kvm2 driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"kvm_network": {
			Type:					schema.TypeString,
			Description:	"The KVM default network name. (kvm2 driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"default",
		},
	
		"kvm_numa_count": {
			Type:					schema.TypeInt,
			Description:	"Simulate numa node count in minikube, supported numa node count range is 1-8 (kvm2 driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	1,
		},
	
		"kvm_qemu_uri": {
			Type:					schema.TypeString,
			Description:	"The KVM QEMU connection URI. (kvm2 driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"qemu:///system",
		},
	
		"listen_address": {
			Type:					schema.TypeString,
			Description:	"IP Address to use to expose ports (docker and podman driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"memory": {
			Type:					schema.TypeString,
			Description:	"Amount of RAM to allocate to Kubernetes (format: <number>[<unit>], where unit = b, k, m or g). Use \"max\" to use the maximum amount of memory. Use \"no-limit\" to not specify a limit (Docker/Podman only))",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"4g",
			StateFunc:	state_utils.MemoryConverter(),
			ValidateDiagFunc:	state_utils.MemoryValidator(),
		},
	
		"mount": {
			Type:					schema.TypeBool,
			Description:	"This will start the mount daemon and automatically mount files into minikube.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"mount_9p_version": {
			Type:					schema.TypeString,
			Description:	"Specify the 9p version that the mount should use",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"9p2000.L",
		},
	
		"mount_gid": {
			Type:					schema.TypeString,
			Description:	"Default group id used for the mount",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"docker",
		},
	
		"mount_ip": {
			Type:					schema.TypeString,
			Description:	"Specify the ip that the mount should be setup on",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"mount_msize": {
			Type:					schema.TypeInt,
			Description:	"The number of bytes to use for 9p packet payload",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	262144,
		},
	
		"mount_options": {
			Type:					schema.TypeSet,
			Description:	"Additional mount options, such as cache=fscache",
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"mount_port": {
			Type:					schema.TypeInt,
			Description:	"Specify the port that the mount should be setup on, where 0 means any free port.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	0,
		},
	
		"mount_string": {
			Type:					schema.TypeString,
			Description:	"The argument to pass the minikube mount command on start.",
			
			Optional:			true,
			ForceNew:			true,
			
			DefaultFunc:	func() (any, error) {
				if runtime.GOOS == "windows" {
					home, err := os.UserHomeDir()
					if err != nil {
						return nil, err
					}
					return home + ":" + "/minikube-host", nil
				} else if runtime.GOOS == "darwin" {
					return "/Users:/minikube-host", nil
				} 
				return "/home:/minikube-host", nil
			},
		},
	
		"mount_type": {
			Type:					schema.TypeString,
			Description:	"Specify the mount filesystem type (supported types: 9p)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"9p",
		},
	
		"mount_uid": {
			Type:					schema.TypeString,
			Description:	"Default user id used for the mount",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"docker",
		},
	
		"namespace": {
			Type:					schema.TypeString,
			Description:	"The named space to activate after start",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"default",
		},
	
		"nat_nic_type": {
			Type:					schema.TypeString,
			Description:	"NIC Type used for nat network. One of Am79C970A, Am79C973, 82540EM, 82543GC, 82545EM, or virtio (virtualbox driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"virtio",
		},
	
		"native_ssh": {
			Type:					schema.TypeBool,
			Description:	"Use native Golang SSH client (default true). Set to 'false' to use the command line 'ssh' command when accessing the docker machine. Useful for the machine drivers when they will not start with 'Waiting for SSH'.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"network": {
			Type:					schema.TypeString,
			Description:	"network to run minikube with. Used by docker/podman, qemu, kvm, and vfkit drivers. If left empty, minikube will create a new network.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"network_plugin": {
			Type:					schema.TypeString,
			Description:	"DEPRECATED: Replaced by --cni",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"nfs_share": {
			Type:					schema.TypeSet,
			Description:	"Local folders to share with Guest via NFS mounts (hyperkit driver only)",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"nfs_shares_root": {
			Type:					schema.TypeString,
			Description:	"Where to root the NFS Shares, defaults to /nfsshares (hyperkit driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"/nfsshares",
		},
	
		"no_kubernetes": {
			Type:					schema.TypeBool,
			Description:	"If set, minikube VM/container will start without starting or configuring Kubernetes. (only works on new clusters)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"no_vtx_check": {
			Type:					schema.TypeBool,
			Description:	"Disable checking for the availability of hardware virtualization before the vm is started (virtualbox driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"nodes": {
			Type:					schema.TypeInt,
			Description:	"The total number of nodes to spin up. Defaults to 1.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	1,
		},
	
		"output": {
			Type:					schema.TypeString,
			Description:	"Format to print stdout in. Options include: [text,json]",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"text",
		},
	
		"ports": {
			Type:					schema.TypeSet,
			Description:	"List of ports that should be exposed (docker and podman driver only)",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"preload": {
			Type:					schema.TypeBool,
			Description:	"If set, download tarball of preloaded images if available to improve start time. Defaults to true.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	true,
		},
	
		"qemu_firmware_path": {
			Type:					schema.TypeString,
			Description:	"Path to the qemu firmware file. Defaults: For Linux, the default firmware location. For macOS, the brew installation location. For Windows, C:\\Program Files\\qemu\\share",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"registry_mirror": {
			Type:					schema.TypeSet,
			Description:	"Registry mirrors to pass to the Docker daemon",
			
			Computed:			true,
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"service_cluster_ip_range": {
			Type:					schema.TypeString,
			Description:	"The CIDR to be used for service cluster IPs.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"10.96.0.0/12",
		},
	
		"socket_vmnet_client_path": {
			Type:					schema.TypeString,
			Description:	"Path to the socket vmnet client binary (QEMU driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			DefaultFunc:	func() (any, error) {
        var prefix string
        if runtime.GOARCH == "arm64" {
            prefix = "/opt/homebrew"
        } else {
            prefix = "/usr/local"
        }
        return prefix + "/opt/socket_vmnet/bin/socket_vmnet_client", nil
    },
		},
	
		"socket_vmnet_path": {
			Type:					schema.TypeString,
			Description:	"Path to socket vmnet binary (QEMU driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			DefaultFunc:	func() (any, error) {
        var prefix string
        if runtime.GOARCH == "arm64" {
            prefix = "/opt/homebrew"
        } else {
            prefix = "/usr/local"
        }
        return prefix + "/var/run/socket_vmnet", nil
    },
		},
	
		"ssh_ip_address": {
			Type:					schema.TypeString,
			Description:	"IP address (ssh driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"ssh_key": {
			Type:					schema.TypeString,
			Description:	"SSH key (ssh driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"ssh_port": {
			Type:					schema.TypeInt,
			Description:	"SSH port (ssh driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	22,
		},
	
		"ssh_user": {
			Type:					schema.TypeString,
			Description:	"SSH user (ssh driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"root",
		},
	
		"static_ip": {
			Type:					schema.TypeString,
			Description:	"Set a static IP for the minikube cluster, the IP must be: private, IPv4, and the last octet must be between 2 and 254, for example 192.168.200.200 (Docker and Podman drivers only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"subnet": {
			Type:					schema.TypeString,
			Description:	"Subnet to be used on kic cluster. If left empty, minikube will choose subnet address, beginning from 192.168.49.0. (docker and podman driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"trace": {
			Type:					schema.TypeString,
			Description:	"Send trace events. Options include: [gcp]",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"uuid": {
			Type:					schema.TypeString,
			Description:	"Provide VM UUID to restore MAC address (hyperkit driver only)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"vm": {
			Type:					schema.TypeBool,
			Description:	"Filter to use only VM Drivers",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	false,
		},
	
		"vm_driver": {
			Type:					schema.TypeString,
			Description:	"DEPRECATED, use `driver` instead.",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	"",
		},
	
		"wait": {
			Type:					schema.TypeSet,
			Description:	"comma separated list of Kubernetes components to verify and wait for after starting a cluster. defaults to \"apiserver,system_pods\", available options: \"apiserver,system_pods,default_sa,apps_running,node_ready,kubelet,extra\" . other acceptable values are 'all' or 'none', 'true' and 'false'",
			
			Optional:			true,
			ForceNew:			true,
			
			Elem: &schema.Schema{
				Type:	schema.TypeString,
			},
			
		},
	
		"wait_timeout": {
			Type:					schema.TypeInt,
			Description:	"max time to wait per Kubernetes or host to be healthy. (Configured in minutes)",
			
			Optional:			true,
			ForceNew:			true,
			
			Default:	6,
		},
	
	}
)

func GetClusterSchema() map[string]*schema.Schema {
	return clusterSchema
}
	