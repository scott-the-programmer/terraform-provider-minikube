//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package service

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/docker/machine/libmachine/ssh"
	"github.com/spf13/viper"
	"k8s.io/klog"
	cmdcfg "k8s.io/minikube/cmd/minikube/cmd/config"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/localpath"
	"k8s.io/minikube/pkg/minikube/node"
	"k8s.io/minikube/pkg/minikube/out/register"

	// Register drivers
	_ "k8s.io/minikube/pkg/minikube/registry/drvs"
)

type ClusterClient interface {
	SetConfig(args MinikubeClientArgs)
	SetDependencies(dep MinikubeClientDeps)
	Start() (*kubeconfig.Settings, error)
	Delete() error
	GetClusterConfig() *config.ClusterConfig
}

type MinikubeClient struct {
	clusterConfig   config.ClusterConfig
	clusterName     string
	addons          []string
	isoUrls         []string
	deleteOnFailure bool
	nodes           int

	// TfCreationLock is a mutex used to prevent multiple minikube clients from conflicting on Start().
	// Only set this if you're using MinikubeClient in a concurrent context
	TfCreationLock *sync.Mutex

	nRunner Node
	dLoader Downloader
}

type MinikubeClientArgs struct {
	ClusterConfig   config.ClusterConfig
	ClusterName     string
	Addons          []string
	IsoUrls         []string
	DeleteOnFailure bool
	Nodes           int
}

type MinikubeClientDeps struct {
	Node       Node
	Downloader Downloader
}

// NewMinikubeClient creates a new MinikubeClient struct
func NewMinikubeClient(args MinikubeClientArgs, dep MinikubeClientDeps) *MinikubeClient {
	return &MinikubeClient{
		clusterConfig:   args.ClusterConfig,
		isoUrls:         args.IsoUrls,
		clusterName:     args.ClusterName,
		addons:          args.Addons,
		deleteOnFailure: args.DeleteOnFailure,
		TfCreationLock:  nil,
		nodes:           args.Nodes,

		nRunner: dep.Node,
		dLoader: dep.Downloader,
	}
}

func init() {
	registerLogging()
	klog.V(klog.Level(1))

	targetDir := localpath.MakeMiniPath("bin")
	new := fmt.Sprintf("%s:%s", targetDir, os.Getenv("PATH"))
	os.Setenv("PATH", new)

	register.Reg.SetStep(register.InitialSetup)

	if runtime.GOOS == "windows" {
		ssh.SetDefaultClient(ssh.Native)
	} else {
		ssh.SetDefaultClient(ssh.External)
	}

}

// SetConfig provides an injection point for setting the cluster config
func (e *MinikubeClient) SetConfig(args MinikubeClientArgs) {
	e.clusterConfig = args.ClusterConfig
	e.isoUrls = args.IsoUrls
	e.clusterName = args.ClusterName
	e.addons = args.Addons
	e.deleteOnFailure = args.DeleteOnFailure
	e.nodes = args.Nodes
}

// SetDependencies injects dependencies into the MinikubeClient
func (e *MinikubeClient) SetDependencies(dep MinikubeClientDeps) {
	e.nRunner = dep.Node
	e.dLoader = dep.Downloader
}

// Start starts the minikube creation process. If the cluster already exists, it will attempt to reuse it
func (e *MinikubeClient) Start() (*kubeconfig.Settings, error) {

	// By nature, viper references (here and within the internals of minikube) are not thread safe.
	// To keep our sanity, let's mutex this call and defer subsequent cluster starts
	if e.TfCreationLock != nil {
		e.TfCreationLock.Lock()
		defer e.TfCreationLock.Unlock()
	}

	viper.Set(cmdcfg.Bootstrapper, "kubeadm")
	viper.Set(config.ProfileName, e.clusterName)
	viper.Set("preload", true)

	url, err := e.downloadIsos()
	if err != nil {
		return nil, err
	}

	e.clusterConfig.MinikubeISO = url

	mRunner, preExists, mAPI, host, err := e.nRunner.Provision(&e.clusterConfig, &e.clusterConfig.Nodes[0], true, true)
	if err != nil {
		return nil, err
	}

	existingAddons := make(map[string]bool)
	for _, addon := range e.addons {
		existingAddons[addon] = true
	}

	starter := node.Starter{
		Runner:         mRunner,
		PreExists:      preExists,
		StopK8s:        false,
		MachineAPI:     mAPI,
		Host:           host,
		Cfg:            &e.clusterConfig,
		Node:           &e.clusterConfig.Nodes[0],
		ExistingAddons: existingAddons,
	}

	kc, err := e.nRunner.Start(starter, true)
	if err != nil {
		return nil, err
	}

	for i := 1; i < e.nodes; i++ {
		err := e.nRunner.Add(&e.clusterConfig, starter)
		if err != nil {
			return kc, err
		}
	}

	klog.Flush()

	return kc, nil
}

// Delete deletes the given cluster associated with the cluster config
func (e *MinikubeClient) Delete() error {
	_, err := e.nRunner.Delete(e.clusterConfig, e.clusterName)
	if err != nil {
		return err
	}
	return nil
}

// GetClusterConfig retrieves the latest cluster config from minikube
func (e *MinikubeClient) GetClusterConfig() *config.ClusterConfig {
	cluster := e.nRunner.Get(e.clusterName)
	return cluster.Config
}

// downloadIsos retrieve all prerequisite images prior to provisioning
func (e *MinikubeClient) downloadIsos() (string, error) {
	url, err := e.dLoader.ISO(e.isoUrls, true)
	if err != nil {
		return "", err
	}

	err = e.dLoader.PreloadTarball(e.clusterConfig.KubernetesConfig.KubernetesVersion,
		e.clusterConfig.KubernetesConfig.ContainerRuntime,
		e.clusterConfig.Driver)
	if err != nil {
		return "", err
	}

	return url, nil
}
