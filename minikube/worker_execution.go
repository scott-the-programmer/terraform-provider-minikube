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

var (
	workerClientOnce sync.Once
	workerClient     *worker.Client
	workerClientErr  error
)

func workerClientInstance() (*worker.Client, error) {
	workerClientOnce.Do(func() {
		workerClient, workerClientErr = worker.NewClient()
	})
	return workerClient, workerClientErr
}

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

func addonTransitions(before, after []string) (enable, disable []string) {
	return lib.DiffAddons(after, before), lib.DiffAddons(before, after)
}

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

func applyAddonsInWorker(ctx context.Context, profile string, enable, disable []string) error {
	_, err := runWorkerOperation(ctx, &worker.Request{
		Op:      worker.OpAddons,
		Profile: profile,
		Enable:  enable,
		Disable: disable,
	})
	return err
}

func deleteClusterInWorker(ctx context.Context, client lib.ClusterClient) error {
	cfg := client.GetConfig()
	_, err := runWorkerOperation(ctx, &worker.Request{
		Op:      worker.OpDelete,
		Profile: cfg.ClusterName,
		Config:  &cfg,
	})
	return err
}

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
