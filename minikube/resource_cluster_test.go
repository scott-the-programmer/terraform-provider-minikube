package minikube

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/state_utils"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/localpath"
)

var _ = flag.String("minikube-start-args", "true", "test") // force minikube into thinking that
// we're running an integration test

type mockClusterClientProperties struct {
	t           *testing.T
	name        string
	haNodes     int
	workerNodes int
	diskSize    int
	memory      string
	cpu         string
}

func TestClusterCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterCreation", 1, 0, 20000, "4096mb", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterConfig("some_driver", "TestClusterCreation"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreation"),
				),
			},
		},
	})
}

func TestClusterUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockUpdate(mockClusterClientProperties{t, "TestClusterUpdate", 1, 0, 20000, "4096mb", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterConfig("some_driver", "TestClusterUpdate"),
			},
			{
				Config: testUnitClusterConfig_Update("some_driver", "TestClusterUpdate"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("minikube_cluster.new", "addons.2", "ingress"),
				),
			},
		},
	})
}

func TestClusterHA(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterCreationHA", 3, 5, 20000, "4096mb", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterHAConfig("some_driver", "TestClusterCreationHA"),
			},
		},
	})
}

func TestClusterDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterCreationDisk", 1, 0, 20480, "4096mb", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterDiskConfig("some_driver", "TestClusterCreationDisk"),
			},
		},
	})
}

func TestClusterWait(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterCreationWait", 1, 0, 20000, "4096mb", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterWaitConfig("some_driver", "TestClusterCreationWait"),
			},
		},
	})
}

func TestClusterCreation_Docker(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig("docker", "TestClusterCreationDocker"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationDocker"),
				),
			},
		},
	})
}

func TestClusterCreation_Docker_Multinode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfigMultinode("docker", "multinode"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "multinode"),
				),
			},
		},
	})
}

func TestClusterCreation_Docker_HA(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfigHighAvailability("docker", "ha"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "ha"),
				),
			},
		},
	})
}

func TestClusterCreation_Docker_ExtraConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterExtraConfig("docker", "TestClusterCreationDocker"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationDocker"),
				),
			},
		},
	})
}

func TestClusterCreation_Docker_Update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig("docker", "TestClusterCreationDockerUpdate"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationDockerUpdate"),
				),
			},
			{
				Config: testAcceptanceClusterConfig_Update("docker", "TestClusterCreationDockerUpdate"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("minikube_cluster.new", "addons.2", "ingress"),
				),
			},
		},
	})
}

func TestClusterCreation_Docker_Addons(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig_StorageProvisioner("docker", "TestClusterCreationDockerAddons"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						err := assertAddonEnabled("TestClusterCreationDockerAddons", "storage-provisioner")
						if err != nil {
							return err
						}
						err = assertAddonEnabled("TestClusterCreationDockerAddons", "dashboard")
						if err != nil {
							return err
						}
						err = assertAddonEnabled("TestClusterCreationDockerAddons", "ingress")
						if err != nil {
							return err
						}
						err = assertAddonEnabled("TestClusterCreationDockerAddons", "default-storageclass")
						if err != nil {
							return err
						}

						return nil
					},
				),
			},
		},
	})
}

func TestClusterCreation_OutOfOrderAddons(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig_OutOfOrderAddons("docker", "TestClusterCreationDocker"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationDocker"),
				),
			},
		},
	})
}

func TestClusterCreation_HAControlPlane(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig_HAControlPlane("docker", "TestClusterCreationDocker"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationDocker"),
				),
			},
		},
	})
}

func TestClusterCreation_Wait(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig_Wait("docker", "TestClusterCreationDocker"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationDocker"),
				),
			},
		},
	})
}

func TestClusterCreation_Hyperkit(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Hyperkit is only supported on macOS")
		return
	}

	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig("hyperkit", "TestClusterCreationHyperkit"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationHyperkit"),
				),
			},
		},
	})
}

func TestClusterCreation_QemuSocketVmNet(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Qemu + SocketVmNet is only supported on macOS")
		return
	}

	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfigQemuSocketVmNet("qemu2", "TestClusterCreationQemu"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationQemu"),
				),
			},
		},
	})
}

func TestClusterCreation_HyperV(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("HyperV is only supported on windows")
		return
	}

	resource.Test(t, resource.TestCase{
		Providers:    map[string]*schema.Provider{"minikube": Provider()},
		CheckDestroy: verifyDelete,
		Steps: []resource.TestStep{
			{
				Config: testAcceptanceClusterConfig("hyperv", "TestClusterCreationHyperV"),
				Check: resource.ComposeTestCheckFunc(
					testPropertyExists("minikube_cluster.new", "TestClusterCreationHyperV"),
				),
			},
		},
	})
}

func TestClusterNoLimitMemory(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterNoLimitMemory", 1, 0, 20000, "no-limit", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterNoLimitMemoryConfig("some_driver", "TestClusterNoLimitMemory"),
			},
		},
	})
}

func TestClusterMaxMemory(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterMaxMemory", 1, 0, 20000, "max", "1"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterMaxMemoryConfig("some_driver", "TestClusterMaxMemory"),
			},
		},
	})
}

func TestClusterNoLimitCPU(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterNoLimitCPU", 1, 0, 20000, "4096mb", "no-limit"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterNoLimitCPUConfig("some_driver", "TestClusterNoLimitCPU"),
			},
		},
	})
}

func TestClusterMaxCPU(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(mockClusterClientProperties{t, "TestClusterMaxCPU", 1, 0, 20000, "4096mb", "max"}))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterMaxCPUConfig("some_driver", "TestClusterMaxCPU"),
			},
		},
	})
}

func mockUpdate(props mockClusterClientProperties) schema.ConfigureContextFunc {
	ctrl := gomock.NewController(props.t)

	mockClusterClient := getBaseMockClient(props.t, ctrl, props.name, props.haNodes, props.workerNodes, props.diskSize, props.memory, props.cpu)

	gomock.InOrder(
		mockClusterClient.EXPECT().
			GetAddons().
			Return([]string{}),
		mockClusterClient.EXPECT().
			GetAddons().
			Return([]string{}),
		mockClusterClient.EXPECT().
			GetAddons().
			Return([]string{}),
		mockClusterClient.
			EXPECT().
			GetAddons().
			Return([]string{
				"dashboard",
				"default-storageclass",
				"ingress",
				"storage-provisioner",
			}),
	)

	configureContext := func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var diags diag.Diagnostics
		mockClusterClientFactory := func() (lib.ClusterClient, error) {
			return mockClusterClient, nil
		}
		return mockClusterClientFactory, diags
	}

	return configureContext
}

func mockSuccess(props mockClusterClientProperties) schema.ConfigureContextFunc {
	ctrl := gomock.NewController(props.t)

	mockClusterClient := getBaseMockClient(props.t, ctrl, props.name, props.haNodes, props.workerNodes, props.diskSize, props.memory, props.cpu)

	mockClusterClient.EXPECT().
		GetAddons().
		Return(nil).
		AnyTimes()

	configureContext := func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var diags diag.Diagnostics
		mockClusterClientFactory := func() (lib.ClusterClient, error) {
			return mockClusterClient, nil
		}
		return mockClusterClientFactory, diags
	}

	return configureContext
}

func getBaseMockClient(t *testing.T, ctrl *gomock.Controller, clusterName string, haNodes int, workerNodes int, diskSize int, memory string, cpu string) *lib.MockClusterClient {
	mockClusterClient := lib.NewMockClusterClient(ctrl)

	os.Mkdir("test_output", 0755)

	d1 := []byte("test contents")
	_ = os.WriteFile("test_output/ca", d1, 0644)
	_ = os.WriteFile("test_output/certificate", d1, 0644)
	_ = os.WriteFile("test_output/key", d1, 0644)

	clusterSchema := ResourceCluster().Schema
	mountString, _ := clusterSchema["mount_string"].DefaultFunc()

	k8sVersion := "v1.26.3"
	kubernetesConfig := config.KubernetesConfig{
		KubernetesVersion:      k8sVersion,
		ClusterName:            clusterName,
		Namespace:              clusterSchema["namespace"].Default.(string),
		APIServerName:          clusterSchema["apiserver_name"].Default.(string),
		APIServerNames:         []string{"minikubeCA"},
		DNSDomain:              clusterSchema["dns_domain"].Default.(string),
		FeatureGates:           clusterSchema["feature_gates"].Default.(string),
		ContainerRuntime:       clusterSchema["container_runtime"].Default.(string),
		CRISocket:              clusterSchema["cri_socket"].Default.(string),
		NetworkPlugin:          clusterSchema["network_plugin"].Default.(string),
		ServiceCIDR:            clusterSchema["service_cluster_ip_range"].Default.(string),
		ImageRepository:        "",
		ShouldLoadCachedImages: clusterSchema["cache_images"].Default.(bool),
		CNI:                    clusterSchema["cni"].Default.(string),
	}

	n := config.Node{
		Name:              "",
		Port:              8443,
		KubernetesVersion: k8sVersion,
		ContainerRuntime:  "docker",
		ControlPlane:      true,
		Worker:            true,
	}

	mem, err := state_utils.GetMemory(memory)
	if err != nil {
		t.Fatalf("Failed to get memory: %v", err)
	}

	c, err := state_utils.GetCPUs(cpu)
	if err != nil {
		t.Fatalf("Failed to get cpu: %v", err)
	}

	cc := config.ClusterConfig{
		Name:                    "terraform-provider-minikube-acc",
		APIServerPort:           clusterSchema["apiserver_port"].Default.(int),
		KeepContext:             clusterSchema["keep_context"].Default.(bool),
		EmbedCerts:              clusterSchema["embed_certs"].Default.(bool),
		MinikubeISO:             defaultIso,
		KicBaseImage:            clusterSchema["base_image"].Default.(string),
		Network:                 clusterSchema["network"].Default.(string),
		Memory:                  mem,
		CPUs:                    c,
		DiskSize:                diskSize,
		Driver:                  "some_driver",
		ListenAddress:           clusterSchema["listen_address"].Default.(string),
		HyperkitVpnKitSock:      clusterSchema["hyperkit_vpnkit_sock"].Default.(string),
		HyperkitVSockPorts:      []string{},
		NFSShare:                []string{},
		NFSSharesRoot:           clusterSchema["nfs_shares_root"].Default.(string),
		DockerEnv:               config.DockerEnv,
		DockerOpt:               config.DockerOpt,
		HostOnlyCIDR:            clusterSchema["host_only_cidr"].Default.(string),
		HypervVirtualSwitch:     clusterSchema["hyperv_virtual_switch"].Default.(string),
		HypervUseExternalSwitch: clusterSchema["hyperv_use_external_switch"].Default.(bool),
		HypervExternalAdapter:   clusterSchema["hyperv_external_adapter"].Default.(string),
		KVMNetwork:              clusterSchema["kvm_network"].Default.(string),
		KVMQemuURI:              clusterSchema["kvm_qemu_uri"].Default.(string),
		KVMGPU:                  clusterSchema["kvm_gpu"].Default.(bool),
		KVMHidden:               clusterSchema["kvm_hidden"].Default.(bool),
		KVMNUMACount:            clusterSchema["kvm_numa_count"].Default.(int),
		DisableDriverMounts:     clusterSchema["disable_driver_mounts"].Default.(bool),
		UUID:                    clusterSchema["uuid"].Default.(string),
		NoVTXCheck:              clusterSchema["no_vtx_check"].Default.(bool),
		DNSProxy:                clusterSchema["dns_proxy"].Default.(bool),
		HostDNSResolver:         clusterSchema["host_dns_resolver"].Default.(bool),
		HostOnlyNicType:         clusterSchema["host_only_nic_type"].Default.(string),
		NatNicType:              clusterSchema["host_only_nic_type"].Default.(string),
		StartHostTimeout:        time.Duration(600 * time.Second),
		ExposedPorts:            []string{},
		SSHIPAddress:            clusterSchema["ssh_ip_address"].Default.(string),
		SSHUser:                 clusterSchema["ssh_user"].Default.(string),
		SSHKey:                  clusterSchema["ssh_key"].Default.(string),
		SSHPort:                 clusterSchema["ssh_port"].Default.(int),
		ExtraDisks:              clusterSchema["extra_disks"].Default.(int),
		CertExpiration:          time.Duration(clusterSchema["cert_expiration"].Default.(int)) * time.Minute,
		Mount:                   clusterSchema["hyperv_use_external_switch"].Default.(bool),
		MountString:             mountString.(string),
		Mount9PVersion:          "9p2000.L",
		MountGID:                "docker",
		MountIP:                 "",
		MountMSize:              262144,
		MountOptions:            []string{},
		MountPort:               0,
		MountType:               "9p",
		MountUID:                "docker",
		BinaryMirror:            "",
		DisableOptimizations:    clusterSchema["hyperv_use_external_switch"].Default.(bool),
		Nodes: []config.Node{
			n,
		},
		KubernetesConfig:   kubernetesConfig,
		MultiNodeRequested: false,
	}

	mockClusterClient.EXPECT().
		SetConfig(gomock.Any()).
		AnyTimes()

	mockClusterClient.EXPECT().
		SetDependencies(gomock.Any()).
		AnyTimes()

	mockClusterClient.EXPECT().
		Start().
		Return(&kubeconfig.Settings{
			ClusterName:          clusterName,
			Namespace:            "default",
			ClusterServerAddress: "http://localhost:8080",
			ClientCertificate:    "test_output/ca",
			CertificateAuthority: "test_output/certificate",
			ClientKey:            "test_output/key",
			KeepContext:          false,
			EmbedCerts:           false,
			ExtensionCluster:     &kubeconfig.Extension{},
			ExtensionContext:     &kubeconfig.Extension{},
		}, nil).
		Times(1)

	mockClusterClient.EXPECT().
		GetClusterConfig().
		Return(&cc).
		AnyTimes()

	mockClusterClient.EXPECT().
		Delete().
		Return(nil)

	mockClusterClient.EXPECT().
		GetK8sVersion().
		Return("v1.99.9").
		AnyTimes()

	mockClusterClient.EXPECT().
		ApplyAddons(gomock.Any()).
		Return(nil).
		AnyTimes()

	mockClusterClient.EXPECT().
		GetConfig().
		Return(lib.MinikubeClientConfig{
			Nodes: workerNodes + haNodes,
			HA:    haNodes > 2,
		}).
		AnyTimes()

	return mockClusterClient
}

func testUnitClusterConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
	}
	`, driver, clusterName)
}

func testUnitClusterDiskConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"

		disk_size = "20g"
	}
	`, driver, clusterName)
}

func testUnitClusterHAConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"

		ha = true
		
		nodes = 8
	}
	`, driver, clusterName)
}

func testUnitClusterWaitConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"

		wait = [ "apiserver" ]
	}
	`, driver, clusterName)
}

func testUnitClusterConfig_Update(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"

		addons = [
			"dashboard",
			"default-storageclass",
			"ingress",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2 
		memory = "6GiB"

		addons = [
			"dashboard",
			"default-storageclass",
			"storage-provisioner",
		]

	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfigQemuSocketVmNet(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"

		network = "socket_vmnet"

		addons = [
			"dashboard",
			"default-storageclass",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfigMultinode(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2
		memory = "6GiB"

		nodes = 3

		addons = [
			"dashboard",
			"default-storageclass",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfigHighAvailability(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 1
		memory = "2GiB"

		nodes = 4
		ha = true

		addons = [
			"dashboard",
			"default-storageclass",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterExtraConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2 
		memory = "6GiB"

		extra_config = ["apiserver.v=3"]
		addons = [
			"dashboard",
			"default-storageclass",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfig_Update(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2 
		memory = "6GiB"

		addons = [
			"dashboard",
			"default-storageclass",
			"ingress",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfig_StorageProvisioner(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2 
		memory = "6000GiB"

		addons = [
			"dashboard",
			"default-storageclass",
			"ingress",
			"storage-provisioner",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfig_OutOfOrderAddons(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2 
		memory = "6000GiB"

		addons = [
			"storage-provisioner",
			"dashboard",
			"ingress",
			"default-storageclass",
		]
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfig_HAControlPlane(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2
		memory = "6000GiB"
		ha = true
	}
	`, driver, clusterName)
}

func testAcceptanceClusterConfig_Wait(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = 2
		memory = "6000GiB"

		wait = [
			"apps_running"
		]
	}
	`, driver, clusterName)
}

func verifyDelete(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "minikube_cluster" {
			continue
		}

		clusterName := rs.Primary.ID
		machineDir := filepath.Join(localpath.MiniPath(), "machines", clusterName)
		profilesDir := filepath.Join(localpath.MiniPath(), "profiles", clusterName)

		_, err := os.Stat(machineDir)
		if err == nil {
			return errors.New("machine dir should not exist")
		}

		_, err = os.Stat(profilesDir)
		if err == nil {
			return errors.New("profiles dir should not exist")
		}
	}

	return nil
}

func assertAddonEnabled(cluster string, addon string) error {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("minikube addons list --profile %s | grep %s", cluster, addon))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	if strings.Contains(string(output), "enabled") {
		return nil
	}

	log.Printf("addon %s not enabled", addon)
	return fmt.Errorf("addon %s not enabled", addon)
}

func testPropertyExists(n string, id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID != id {
			return fmt.Errorf("No cluster id set")
		}

		return nil
	}
}

func testUnitClusterNoLimitMemoryConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		memory = "no-limit"
	}
	`, driver, clusterName)
}

func testUnitClusterMaxMemoryConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		memory = "max"
	}
	`, driver, clusterName)
}

func testUnitClusterNoLimitCPUConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = "no-limit"
	}
	`, driver, clusterName)
}

func testUnitClusterMaxCPUConfig(driver string, clusterName string) string {
	return fmt.Sprintf(`
	resource "minikube_cluster" "new" {
		driver = "%s"
		cluster_name = "%s"
		cpus = "max"
	}
	`, driver, clusterName)
}
