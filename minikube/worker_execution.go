package minikube

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/worker"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
)

// The provider re-executes itself for every lifecycle operation so embedded
// Minikube's process-global configuration state (Viper singletons set and read
// by node.Start/Provision) cannot leak between concurrently created clusters.
// See minikube/worker for the protocol; the helpers below are the only wiring
// the resource code needs. Setting TF_MINIKUBE_IN_PROCESS reverts to the
// previous in-process behavior, where the broad creation mutex serializes
// everything.

var (
	workerClientOnce sync.Once
	workerClient     *worker.Client
	workerClientErr  error
)

// workerClientInstance resolves the parent handle once per provider process.
func workerClientInstance() (*worker.Client, error) {
	workerClientOnce.Do(func() {
		workerClient, workerClientErr = worker.NewClient()
	})
	return workerClient, workerClientErr
}

// runWorkerOperation executes one lifecycle step in a fresh worker process,
// mapping protocol failures to plain errors for the Terraform diagnostics.
// Operations on the same profile serialize inside the worker package; distinct
// profiles overlap.
func runWorkerOperation(ctx context.Context, req *worker.Request) (*worker.Response, error) {
	client, err := workerClientInstance()
	if err != nil {
		return nil, err
	}
	resp, err := client.Run(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp, nil
}

// enabledAddons lists the addons currently enabled on a saved cluster
// configuration, mirroring MinikubeClient.GetAddons without a second profile
// load.
func enabledAddons(cc *config.ClusterConfig) []string {
	if cc == nil {
		return []string{}
	}
	addons := make([]string, 0, len(cc.Addons))
	for addon, enabled := range cc.Addons {
		if enabled {
			addons = append(addons, addon)
		}
	}
	return addons
}

// addonTransitions expresses an addon change as explicit enable/disable lists,
// which is what the worker protocol requires because a fresh worker observes
// no request history.
func addonTransitions(before, after []string) (enable, disable []string) {
	return lib.DiffAddons(after, before), lib.DiffAddons(before, after)
}

// createClusterInWorker runs MinikubeClient.Start in a fresh worker process.
// The worker performs downloads, provisioning, startup, additional nodes, and
// addon application exactly as the in-process path would, but against its own
// process-global Minikube state; the kubeconfig subset the parent's outputs
// need travels back over the response frame.
func createClusterInWorker(ctx context.Context, client lib.ClusterClient) (*kubeconfig.Settings, error) {
	cfg := client.GetConfig()
	resp, err := runWorkerOperation(ctx, &worker.Request{
		Op:      worker.OpStart,
		Profile: cfg.ClusterName,
		Config:  &cfg,
	})
	if err != nil {
		return nil, err
	}
	out := resp.Kubeconfig
	if out == nil {
		return nil, fmt.Errorf("worker start response for %q missing kubeconfig outputs", cfg.ClusterName)
	}
	return &kubeconfig.Settings{
		ClusterName:          out.ClusterName,
		ClusterServerAddress: out.ClusterServerAddress,
		ClientKey:            out.ClientKey,
		ClientCertificate:    out.ClientCertificate,
		CertificateAuthority: out.CertificateAuthority,
	}, nil
}

// applyAddonsInWorker toggles addons on an existing profile in a fresh worker.
// The enable/disable lists are computed in the parent because addon state
// transitions cannot be re-derived inside a single-request worker.
func applyAddonsInWorker(ctx context.Context, profile string, enable, disable []string) error {
	_, err := runWorkerOperation(ctx, &worker.Request{
		Op:      worker.OpAddons,
		Profile: profile,
		Enable:  enable,
		Disable: disable,
	})
	return err
}

// deleteClusterInWorker destroys a profile and its machine directories in a
// fresh worker.
func deleteClusterInWorker(ctx context.Context, client lib.ClusterClient) error {
	cfg := client.GetConfig()
	_, err := runWorkerOperation(ctx, &worker.Request{
		Op:      worker.OpDelete,
		Profile: cfg.ClusterName,
		Config:  &cfg,
	})
	return err
}

// readClusterInWorker loads the saved profile configuration in a fresh worker
// and returns it with its enabled addons for refresh.
func readClusterInWorker(ctx context.Context, profile string) (*config.ClusterConfig, []string, error) {
	resp, err := runWorkerOperation(ctx, &worker.Request{
		Op:      worker.OpGet,
		Profile: profile,
	})
	if err != nil {
		return nil, nil, err
	}
	cc := resp.ClusterConfig
	if cc == nil {
		return nil, nil, fmt.Errorf("worker refresh response for %q missing cluster configuration", profile)
	}
	return cc, enabledAddons(cc), nil
}
