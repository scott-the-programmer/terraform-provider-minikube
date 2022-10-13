//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package service

import (
	"os"
	"path/filepath"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/host"
	"k8s.io/minikube/pkg/minikube/command"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/exit"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/localpath"
	"k8s.io/minikube/pkg/minikube/mustload"
	"k8s.io/minikube/pkg/minikube/node"
	"k8s.io/minikube/pkg/minikube/reason"
)

type Node interface {
	Provision(cc *config.ClusterConfig, n *config.Node, apiServer bool, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error)
	Start(starter node.Starter, apiServer bool) (*kubeconfig.Settings, error)
	Delete(cc config.ClusterConfig, name string) (*config.Node, error)
	Get(name string) mustload.ClusterController
	Add(cc *config.ClusterConfig, starter node.Starter) error
}

type MinikubeNode struct {
	workerNodes int
}

func NewMinikubeNode() *MinikubeNode {
	return &MinikubeNode{workerNodes: 0}
}

func (m *MinikubeNode) Provision(cc *config.ClusterConfig, n *config.Node, apiServer bool, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error) {
	makeAllMinikubeDirectories()
	_, err := node.CacheKubectlBinary(cc.KubernetesConfig.KubernetesVersion, cc.BinaryMirror)
	if err != nil {
		return nil, false, nil, nil, err
	}

	return node.Provision(cc, n, apiServer, delOnFail)
}

func (m *MinikubeNode) Start(starter node.Starter, apiServer bool) (*kubeconfig.Settings, error) {

	return node.Start(starter, apiServer)
}

//Add adds nodes to the clusters node pool
func (m *MinikubeNode) Add(cc *config.ClusterConfig, starter node.Starter) error {
	n := config.Node{
		Name:              node.Name(m.workerNodes),
		Worker:            true,
		ControlPlane:      false,
		KubernetesVersion: starter.Cfg.KubernetesConfig.KubernetesVersion,
		ContainerRuntime:  starter.Cfg.KubernetesConfig.ContainerRuntime,
	}
	m.workerNodes++
	return node.Add(cc, n, true)
}

func (m *MinikubeNode) Delete(cc config.ClusterConfig, name string) (*config.Node, error) {
	node, err := node.Delete(cc, name)
	if err != nil {
		return node, err
	}

	machineDir := filepath.Join(localpath.MiniPath(), "machines", name)
	profilesDir := filepath.Join(localpath.MiniPath(), "profiles", name)
	err = rmdir(machineDir)
	if err != nil {
		return node, err
	}

	err = rmdir(profilesDir)
	if err != nil {
		return node, err
	}

	return node, err
}

func (m *MinikubeNode) Get(name string) mustload.ClusterController {
	return mustload.Running(name)
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
