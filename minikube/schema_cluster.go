//go:generate go run ../generate/main.go -target $GOFILE
package minikube

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

var (
	clusterSchema = map[string]*schema.Schema{
		"cluster_name": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The name of the minikube cluster",
			Default:     "terraform-provider-minikube",
		},

		"addons": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "Enable addons. see `minikube addons list` for a list of valid addon names.",
		},

		"apiserver_ips": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "A set of apiserver IP Addresses which are used in the generated certificate for kubernetes.  This can be used if you want to make the apiserver available from outside the machine",
		},

		"apiserver_name": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The authoritative apiserver hostname for apiserver certificates and connectivity. This can be used if you want to make the apiserver available from outside the machine",
			Default:     "minikubeCA",
		},

		"apiserver_names": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type:    schema.TypeString,
				Default: "minikubeCA",
			},
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "A set of apiserver names which are used in the generated certificate for kubernetes.  This can be used if you want to make the apiserver available from outside the machine",
		},

		"apiserver_port": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "The apiserver listening port",
			Default:     8443,
		},

		"base_image": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The base image to use for docker/podman drivers. Intended for local development.",
			Default:     "gcr.io/k8s-minikube/kicbase:v0.0.33@sha256:73b259e144d926189cf169ae5b46bbec4e08e4e2f2bd87296054c3244f70feb8",
		},

		"cache_images": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "If true, cache docker images for the current bootstrapper and load them into the machine. Always false with --driver=none.",
			Default:     true,
		},

		"cert_expiration": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Duration (in hours) until minikube certificate expiration, defaults to three years (26280h).",
			Default:     26280,
		},

		"cni": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "CNI plug-in to use. Valid options  auto, bridge, calico, cilium, flannel, kindnet, or path to a CNI manifest (default  auto)",
			Default:     "",
		},

		"container_runtime": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The container runtime to be used (docker, cri-o, containerd).",
			Default:     "docker",
		},

		"cpus": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Number of CPUs allocated to Kubernetes. Use \"max\"to use the maximum number of CPUs.",
			Default:     2,
		},

		"cri_socket": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The cri socket path to be used.",
			Default:     "",
		},

		"delete_on_failure": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "If set, delete the current cluster if start fails and try again. Defaults to false.",
			Default:     false,
		},

		"disable_driver_mounts": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Disables the filesystem mounts provided by the hypervisors",
			Default:     false,
		},

		"disk_size": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Disk size allocated to the minikube VM in mb",
			Default:     20000,
		},

		"dns_domain": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The cluster dns domain name used in the Kubernetes cluster",
			Default:     "cluster.local",
		},

		"dns_proxy": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Enable proxy for NAT DNS requests (virtualbox driver only)",
			Default:     false,
		},

		"driver": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Driver is one of  virtualbox, parallels, vmwarefusion, hyperkit, vmware, docker, podman (experimental), ssh (defaults to auto-detect)",
			Default:     "docker",
		},

		"embed_certs": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "if true, will embed the certs in kubeconfig.",
			Default:     false,
		},

		"extra_config": {
			Type:        schema.TypeMap,
			Optional:    true,
			ForceNew:    true,
			Description: "A set of key=value pairs that describe configuration that may be passed to different components.",
			Default:     "",
		},

		"extra_disks": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Number of extra disks created and attached to the minikube VM (currently only implemented for hyperkit and kvm2 drivers)",
			Default:     0,
		},

		"feature_gates": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "A set of key=value pairs that describe feature gates for alpha/experimental features.",
			Default:     "",
		},

		"force": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Force minikube to perform possibly dangerous operations",
			Default:     false,
		},

		"host_dns_resolver": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Enable host resolver for NAT DNS requests (virtualbox driver only)",
			Default:     true,
		},

		"host_only_cidr": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The CIDR to be used for the minikube VM (virtualbox driver only)",
			Default:     "192.168.59.1/24",
		},

		"host_only_nic_type": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "NIC Type used for host only network. One of Am79C970A, Am79C973, 82540EM, 82543GC, 82545EM, or virtio (virtualbox driver only)",
			Default:     "virtio",
		},

		"hyperkit_vpnkit_sock": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Location of the VPNKit socket used for networking. If empty, disables Hyperkit VPNKitSock, if 'auto' uses Docker for Mac VPNKit connection, otherwise uses the specified VSock (hyperkit driver only)",
			Default:     "",
		},

		"hyperkit_vsock_ports": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "List of guest VSock ports that should be exposed as sockets on the host (hyperkit driver only)",
		},

		"hyperv_external_adapter": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "External Adapter on which external switch will be created if no external switch is found. (hyperv driver only)",
			Default:     "",
		},

		"hyperv_use_external_switch": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Whether to use external switch over Default Switch if virtual switch not explicitly specified. (hyperv driver only)",
			Default:     false,
		},

		"hyperv_virtual_switch": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The hyperv virtual switch name. Defaults to first found. (hyperv driver only)",
			Default:     "",
		},

		"image_mirror_country": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Country code of the image mirror to be used. Leave empty to use the global one. For Chinese mainland users, set it to cn.",
			Default:     "",
		},

		"image_repository": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Alternative image repository to pull docker images from. This can be used when you have limited access to gcr.io. Set it to \"auto\"to let minikube decide one for you. For Chinese mainland users, you may use local gcr.io mirrors such as registry.cn-hangzhou.aliyuncs.com/google_containers",
			Default:     "",
		},

		"insecure_registry": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "Insecure Docker registries to pass to the Docker daemon.  The default service CIDR range will automatically be added.",
		},

		"iso_url": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "Locations to fetch the minikube ISO from.",
		},

		"keep_context": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "This will keep the existing kubectl context and will create a minikube context.",
			Default:     false,
		},

		"kvm_gpu": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Enable experimental NVIDIA GPU support in minikube",
			Default:     false,
		},

		"kvm_hidden": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Hide the hypervisor signature from the guest in minikube (kvm2 driver only)",
			Default:     false,
		},

		"kvm_network": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The KVM default network name. (kvm2 driver only)",
			Default:     "default",
		},

		"kvm_numa_count": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Simulate numa node count in minikube, supported numa node count range is 1-8 (kvm2 driver only)",
			Default:     1,
		},

		"kvm_qemu_uri": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The KVM QEMU connection URI. (kvm2 driver only)",
			Default:     "qemu:///system",
		},

		"listen_address": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "IP Address to use to expose ports (docker and podman driver only)",
			Default:     "",
		},

		"memory": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Amount of RAM to allocate to Kubernetes in mb",
			Default:     6000,
		},

		"mount": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "This will start the mount daemon and automatically mount files into minikube.",
			Default:     false,
		},

		"mount_string": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "/minikube-host'  The argument to pass the minikube mount command on start.",
			Default:     "/Users",
		},

		"namespace": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The named space to activate after start",
			Default:     "default",
		},

		"nat_nic_type": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "NIC Type used for nat network. One of Am79C970A, Am79C973, 82540EM, 82543GC, 82545EM, or virtio (virtualbox driver only)",
			Default:     "virtio",
		},

		"network": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "network to run minikube with. Now it is used by docker/podman and KVM drivers. If left empty, minikube will create a new network.",
			Default:     "",
		},

		"network_plugin": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Kubelet network plug-in to use (default  auto)",
			Default:     "",
		},

		"nfs_share": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "Local folders to share with Guest via NFS mounts (hyperkit driver only)",
		},

		"nfs_shares_root": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Where to root the NFS Shares, defaults to /nfsshares (hyperkit driver only)",
			Default:     "/nfsshares",
		},

		"no_vtx_check": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Disable checking for the availability of hardware virtualization before the vm is started (virtualbox driver only)",
			Default:     false,
		},

		"nodes": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "The number of nodes to spin up. Defaults to 1.",
			Default:     1,
		},

		"ports": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "List of ports that should be exposed (docker and podman driver only)",
		},

		"registry_mirror": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "Registry mirrors to pass to the Docker daemon",
		},

		"service_cluster_ip_range": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The CIDR to be used for service cluster IPs.",
			Default:     "10.96.0.0/12",
		},

		"ssh_ip_address": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "IP address (ssh driver only)",
			Default:     "",
		},

		"ssh_key": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "SSH key (ssh driver only)",
			Default:     "",
		},

		"ssh_port": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "SSH port (ssh driver only)",
			Default:     22,
		},

		"ssh_user": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "SSH user (ssh driver only)",
			Default:     "root",
		},

		"uuid": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Provide VM UUID to restore MAC address (hyperkit driver only)",
			Default:     "",
		},

		"wait_timeout": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "max time (in seconds) to wait per Kubernetes or host to be healthy.",
			Default:     600,
		},

		"vm": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Filter to use only VM Drivers",
			Default:     false,
		},

		"vm_driver": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "DEPRECATED, use `driver` instead.",
			Default:     "",
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
	}
)

func GetClusterSchema() map[string]*schema.Schema {
	return clusterSchema
}
