//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock_minikube_cluster.go -package=$GOPACKAGE
package lib

import (
	"os"
	"path/filepath"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/host"
	delete "k8s.io/minikube/cmd/minikube/cmd"
	minikubeAddons "k8s.io/minikube/pkg/addons"
	"k8s.io/minikube/pkg/minikube/command"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/exit"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/localpath"
	"k8s.io/minikube/pkg/minikube/mustload"
	"k8s.io/minikube/pkg/minikube/node"
	"k8s.io/minikube/pkg/minikube/reason"
)

type Cluster interface {
	Provision(cc *config.ClusterConfig, n *config.Node, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error)
	Start(starter node.Starter) (*kubeconfig.Settings, error)
	Delete(cc config.ClusterConfig, name string) (*config.Node, error)
	Get(name string) *config.ClusterConfig
	AddWorkerNode(cc *config.ClusterConfig, starter node.Starter) error
	AddControlPlaneNode(cc *config.ClusterConfig, starter node.Starter) error
	SetAddon(name string, addon string, value string) error
}

type MinikubeCluster struct {
	nodes int
}

func NewMinikubeCluster() *MinikubeCluster {
	return &MinikubeCluster{nodes: 0}
}

func (m *MinikubeCluster) Provision(cc *config.ClusterConfig, n *config.Node, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error) {
	makeAllMinikubeDirectories()
	_, err := node.CacheKubectlBinary(cc.KubernetesConfig.KubernetesVersion, cc.BinaryMirror)
	if err != nil {
		return nil, false, nil, nil, err
	}

	r, s, l, h, err := node.Provision(cc, n, delOnFail)
	if err != nil {
		return nil, false, nil, nil, err
	}
	m.nodes++
	return r, s, l, h, err
}

func (m *MinikubeCluster) Start(starter node.Starter) (*kubeconfig.Settings, error) {
	s, err := node.Start(starter)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// AddWorkerNode adds a new worker node to the clusters node pool
func (m *MinikubeCluster) AddWorkerNode(cc *config.ClusterConfig, starter node.Starter) error {
	m.nodes++
	n := config.Node{
		// index starts from 1 https://github.com/kubernetes/minikube/blob/075c1b01f2f8778ac746e05098044234a3f0b06f/pkg/minikube/driver/driver.go#L387C4-L387C27
		Name:              node.Name(m.nodes),
		Worker:            true,
		ControlPlane:      false,
		KubernetesVersion: starter.Cfg.KubernetesConfig.KubernetesVersion,
		ContainerRuntime:  starter.Cfg.KubernetesConfig.ContainerRuntime,
	}
	return node.Add(cc, n, true)
}

// AddControlPanelNode adds a new control panel node to the clusters node pool
// useful for provisioning high availability clusters
func (m *MinikubeCluster) AddControlPlaneNode(cc *config.ClusterConfig, starter node.Starter) error {
	m.nodes++
	n := config.Node{
		Name:              node.Name(m.nodes),
		Worker:            true,
		ControlPlane:      true,
		KubernetesVersion: starter.Cfg.KubernetesConfig.KubernetesVersion,
		ContainerRuntime:  starter.Cfg.KubernetesConfig.ContainerRuntime,
	}
	return node.Add(cc, n, true)
}

func (m *MinikubeCluster) Delete(cc config.ClusterConfig, name string) (*config.Node, error) {
	errs := delete.DeleteProfiles([]*config.Profile{
		{
			Name:   name,
			Config: &cc,
		},
	})
	if len(errs) > 0 {
		return nil, errs[0]
	}

	machineDir := filepath.Join(localpath.MiniPath(), "machines", name)
	profilesDir := filepath.Join(localpath.MiniPath(), "profiles", name)
	err := rmdir(machineDir)
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
	return minikubeAddons.SetAndSave(name, addon, value)
}

func (m *MinikubeCluster) Get(name string) *config.ClusterConfig {
	_, config := mustload.Partial(name)
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
