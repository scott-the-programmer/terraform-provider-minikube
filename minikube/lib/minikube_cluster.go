//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock_minikube_cluster.go -package=$GOPACKAGE
package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	delete "k8s.io/minikube/cmd/minikube/cmd"
	minikubeAddons "k8s.io/minikube/pkg/addons"
	"k8s.io/minikube/pkg/libmachine"
	"k8s.io/minikube/pkg/libmachine/host"
	"k8s.io/minikube/pkg/minikube/assets"
	"k8s.io/minikube/pkg/minikube/command"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/exit"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/localpath"
	"k8s.io/minikube/pkg/minikube/mustload"
	"k8s.io/minikube/pkg/minikube/node"
	"k8s.io/minikube/pkg/minikube/reason"
	"k8s.io/minikube/pkg/minikube/run"
)

type Cluster interface {
	Provision(cc *config.ClusterConfig, n *config.Node, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error)
	Start(starter node.Starter) (*kubeconfig.Settings, error)
	Delete(cc *config.ClusterConfig, name string) (*config.Node, error)
	Get(name string) *config.ClusterConfig
	AddWorkerNode(cc *config.ClusterConfig, kv string, apiServerPort int, cr string) error
	AddControlPlaneNode(cc *config.ClusterConfig, k8sVersion string, port int, containerRuntime string) (*config.ClusterConfig, error)
	SetAddon(name string, addon string, value string) error
}

type MinikubeCluster struct {
	nodes          int
	commandOptions *run.CommandOptions
}

func NewMinikubeCluster() *MinikubeCluster {
	return &MinikubeCluster{
		nodes: 0,
		commandOptions: &run.CommandOptions{
			NonInteractive: true,
			DownloadOnly:   false,
		},
	}
}

func (m *MinikubeCluster) Provision(cc *config.ClusterConfig, n *config.Node, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error) {
	makeAllMinikubeDirectories()
	_, err := node.CacheKubectlBinary(cc.KubernetesConfig.KubernetesVersion, cc.BinaryMirror)
	if err != nil {
		return nil, false, nil, nil, err
	}

	r, s, l, h, err := node.Provision(cc, n, delOnFail, m.commandOptions)
	if err != nil {
		return nil, false, nil, nil, err
	}
	m.nodes++
	return r, s, l, h, err
}

func (m *MinikubeCluster) Start(starter node.Starter) (*kubeconfig.Settings, error) {
	if err := resetAddonAssets(); err != nil {
		return nil, err
	}
	s, err := node.Start(starter, m.commandOptions)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (m *MinikubeCluster) AddControlPlaneNode(cc *config.ClusterConfig, k8sVersion string, port int, containerRuntime string) (*config.ClusterConfig, error) {
	n := config.Node{
		Name:              node.Name(m.nodes + 1),
		Worker:            true,
		ControlPlane:      true,
		KubernetesVersion: k8sVersion,
		Port:              port,
		ContainerRuntime:  containerRuntime,
	}

	err := node.Add(cc, n, false, m.commandOptions)
	if err != nil {
		return nil, err
	}

	m.nodes++
	return cc, nil
}

// AddWorkerNode adds a new worker node to the clusters node pool
func (m *MinikubeCluster) AddWorkerNode(cc *config.ClusterConfig, kv string, apiServerPort int, cr string) error {
	m.nodes++
	n := config.Node{
		// index starts from 1 https://github.com/kubernetes/minikube/blob/075c1b01f2f8778ac746e05098044234a3f0b06f/pkg/minikube/driver/driver.go#L387C4-L387C27
		Name:              node.Name(m.nodes),
		Worker:            true,
		ControlPlane:      false,
		KubernetesVersion: kv,
		Port:              apiServerPort,
		ContainerRuntime:  cr,
	}
	return node.Add(cc, n, true, m.commandOptions)
}

func (m *MinikubeCluster) Delete(cc *config.ClusterConfig, name string) (*config.Node, error) {
	// Terraform's reconstructed config only contains the primary node. Use the
	// saved node inventory so Minikube also removes secondary nodes' volumes.
	saved, err := config.Load(name)
	if err == nil {
		cc = saved
	} else if !config.IsNotExist(err) {
		return nil, fmt.Errorf("load cluster for deletion: %w", err)
	}
	errs := delete.DeleteProfiles([]*config.Profile{
		{
			Name:   name,
			Config: cc,
		},
	}, nil)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	machineDir := filepath.Join(localpath.MiniPath(), "machines", name)
	profilesDir := filepath.Join(localpath.MiniPath(), "profiles", name)
	err = rmdir(machineDir)
	if err != nil {
		return nil, err
	}

	err = rmdir(profilesDir)
	if err != nil {
		return nil, err
	}

	return nil, err
}

func (m *MinikubeCluster) SetAddon(name string, addon string, value string) error {
	if err := resetAddonAssets(); err != nil {
		return err
	}
	return minikubeAddons.SetAndSave(name, addon, value, m.commandOptions)
}

// Minikube keeps bundled addon readers globally. Unlike the CLI, a provider
// process can install them repeatedly, so rewind them under the creation lock.
func resetAddonAssets() error {
	for _, addon := range assets.Addons {
		for _, asset := range addon.Assets {
			if _, err := asset.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("rewind addon asset %s: %w", asset.GetSourcePath(), err)
			}
		}
	}
	return nil
}

func (m *MinikubeCluster) Get(name string) *config.ClusterConfig {
	_, config := mustload.Partial(name, nil)
	return config
}

func makeAllMinikubeDirectories() {
	dirs := [...]string{
		localpath.MakeMiniPath("certs"),
		localpath.MakeMiniPath("machines"),
		localpath.MakeMiniPath("cache"),
		localpath.MakeMiniPath("config"),
		localpath.MakeMiniPath("addons"),
		localpath.MakeMiniPath("files"),
		localpath.MakeMiniPath("logs"),
	}
	for _, path := range dirs {
		if err := os.MkdirAll(path, 0777); err != nil {
			exit.Error(reason.HostHomeMkdir, "Error creating minikube directory", err)
		}
	}
}

func rmdir(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}

	return nil
}
