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

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/service"

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

func TestClusterCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockSuccess(t, "TestClusterCreation"))},
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
		Providers:  map[string]*schema.Provider{"minikube": NewProvider(mockUpdate(t, "TestClusterCreation"))},
		Steps: []resource.TestStep{
			{
				Config: testUnitClusterConfig("some_driver", "TestClusterCreation"),
			},
			{
				Config: testUnitClusterConfig_Update("some_driver", "TestClusterCreation"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("minikube_cluster.new", "addons.2", "ingress"),
				),
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

func mockUpdate(t *testing.T, clusterName string) schema.ConfigureContextFunc {
	ctrl := gomock.NewController(t)

	mockClusterClient := getBaseMockClient(ctrl, clusterName)

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
		mockClusterClientFactory := func() (service.ClusterClient, error) {
			return mockClusterClient, nil
		}
		return mockClusterClientFactory, diags
	}

	return configureContext
}

func mockSuccess(t *testing.T, clusterName string) schema.ConfigureContextFunc {
	ctrl := gomock.NewController(t)

	mockClusterClient := getBaseMockClient(ctrl, clusterName)

	mockClusterClient.EXPECT().
		GetAddons().
		Return(nil).
		AnyTimes()

	configureContext := func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var diags diag.Diagnostics
		mockClusterClientFactory := func() (service.ClusterClient, error) {
			return mockClusterClient, nil
		}
		return mockClusterClientFactory, diags
	}

	return configureContext
}

func getBaseMockClient(ctrl *gomock.Controller, clusterName string) *service.MockClusterClient {
	mockClusterClient := service.NewMockClusterClient(ctrl)

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
		APIServerNames:         []string{clusterSchema["apiserver_name"].Default.(string)},
		DNSDomain:              clusterSchema["dns_domain"].Default.(string),
		FeatureGates:           clusterSchema["feature_gates"].Default.(string),
		ContainerRuntime:       clusterSchema["container_runtime"].Default.(string),
		CRISocket:              clusterSchema["cri_socket"].Default.(string),
		NetworkPlugin:          clusterSchema["network_plugin"].Default.(string),
		ServiceCIDR:            clusterSchema["service_cluster_ip_range"].Default.(string),
		ImageRepository:        "",
		ShouldLoadCachedImages: clusterSchema["cache_images"].Default.(bool),
		CNI:                    clusterSchema["cni"].Default.(string),
		NodePort:               clusterSchema["apiserver_port"].Default.(int),
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
		KeepContext:             clusterSchema["keep_context"].Default.(bool),
		EmbedCerts:              clusterSchema["embed_certs"].Default.(bool),
		MinikubeISO:             defaultIso,
		KicBaseImage:            clusterSchema["base_image"].Default.(string),
		Network:                 clusterSchema["network"].Default.(string),
		Memory:                  4000,
		CPUs:                    2,
		DiskSize:                20000,
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
		Return(service.MinikubeClientConfig{}).
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
		memory = "6000mb"

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
		memory = "6000mb"

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
		memory = "6000mb"

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
		memory = "6000mb"

		addons = [
			"storage-provisioner",
			"dashboard",
			"ingress",
			"default-storageclass",
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
