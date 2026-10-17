package minikube

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/worker"
	"github.com/stretchr/testify/require"
	"k8s.io/minikube/pkg/minikube/config"
)

// The child checks the whole request before returning the supplied response.
func expectWorkerRequest(t *testing.T, req worker.Request, resp worker.Response) {
	t.Helper()
	client, err := workerClientInstance()
	require.NoError(t, err)
	saved := *client
	t.Cleanup(func() { *client = saved })
	payload, err := json.Marshal(req)
	require.NoError(t, err)
	response, err := json.Marshal(resp)
	require.NoError(t, err)
	client.ExtraEnv = append([]string{}, client.ExtraEnv...)
	client.ExtraEnv = append(client.ExtraEnv,
		"TF_MINIKUBE_PROVIDER_TEST_REQUEST="+string(payload),
		"TF_MINIKUBE_PROVIDER_TEST_RESPONSE="+string(response))
	t.Setenv("MINIKUBE_HOME", t.TempDir())
}

func TestWorkerLifecycleRequests(t *testing.T) {
	cfg := lib.MinikubeClientConfig{
		ClusterName: "alpha", ClusterConfig: &config.ClusterConfig{Name: "alpha", CPUs: 2, Memory: 4096},
		Addons: []string{"ingress"}, IsoUrls: []string{"file:///minikube.iso"},
		Nodes: 3, HA: true, NativeSsh: true, DeleteOnFailure: true,
	}
	client := lib.NewMinikubeClient(cfg, lib.MinikubeClientDeps{})
	ctx := context.Background()
	t.Run("start", func(t *testing.T) {
		out := &worker.KubeconfigOutputs{
			ClusterName: "alpha", ClusterServerAddress: "https://127.0.0.1:8443",
			ClientKey: "/key", ClientCertificate: "/cert", CertificateAuthority: "/ca",
		}
		expectWorkerRequest(t, worker.Request{Op: worker.OpStart, Profile: "alpha", Config: &cfg},
			worker.Response{Op: worker.OpStart, OK: true, Kubeconfig: out})
		got, err := createClusterInWorker(ctx, client)
		require.NoError(t, err)
		require.Equal(t, out.Settings(), got)
	})
	t.Run("addons", func(t *testing.T) {
		enable, disable := addonTransitions([]string{"keep", "old"}, []string{"keep", "new"})
		require.Equal(t, []string{"new"}, enable)
		require.Equal(t, []string{"old"}, disable)
		expectWorkerRequest(t, worker.Request{Op: worker.OpAddons, Profile: "alpha", Enable: enable, Disable: disable},
			worker.Response{Op: worker.OpAddons, OK: true})
		require.NoError(t, applyAddonsInWorker(ctx, "alpha", enable, disable))
	})
	t.Run("delete", func(t *testing.T) {
		expectWorkerRequest(t, worker.Request{Op: worker.OpDelete, Profile: "alpha", Config: &cfg},
			worker.Response{Op: worker.OpDelete, OK: true})
		require.NoError(t, deleteClusterInWorker(ctx, client))
	})
	t.Run("refresh", func(t *testing.T) {
		cc := &config.ClusterConfig{Name: "alpha", Addons: map[string]bool{"ingress": true, "disabled": false}}
		expectWorkerRequest(t, worker.Request{Op: worker.OpGet, Profile: "alpha"},
			worker.Response{Op: worker.OpGet, OK: true, ClusterConfig: cc})
		got, addons, err := readClusterInWorker(ctx, "alpha")
		require.NoError(t, err)
		require.Equal(t, cc, got)
		require.Equal(t, []string{"ingress"}, addons)
	})
	require.Empty(t, enabledAddons(nil))
}

func TestWorkerLifecycleErrors(t *testing.T) {
	cfg := lib.MinikubeClientConfig{ClusterName: "alpha"}
	client := lib.NewMinikubeClient(cfg, lib.MinikubeClientDeps{})
	ctx := context.Background()
	for _, op := range []string{worker.OpStart, worker.OpAddons, worker.OpDelete, worker.OpGet} {
		t.Run(op, func(t *testing.T) {
			req := worker.Request{Op: op, Profile: "alpha"}
			if op == worker.OpStart || op == worker.OpDelete {
				req.Config = &cfg
			}
			expectWorkerRequest(t, req, worker.Response{Op: op, Error: "operation failed"})
			var err error
			switch op {
			case worker.OpStart:
				_, err = createClusterInWorker(ctx, client)
			case worker.OpAddons:
				err = applyAddonsInWorker(ctx, "alpha", nil, nil)
			case worker.OpDelete:
				err = deleteClusterInWorker(ctx, client)
			case worker.OpGet:
				_, _, err = readClusterInWorker(ctx, "alpha")
			}
			require.EqualError(t, err, "operation failed")
		})
	}
	t.Run("missing start outputs", func(t *testing.T) {
		expectWorkerRequest(t, worker.Request{Op: worker.OpStart, Profile: "alpha", Config: &cfg},
			worker.Response{Op: worker.OpStart, OK: true})
		_, err := createClusterInWorker(ctx, client)
		require.ErrorContains(t, err, "missing kubeconfig outputs")
	})
	t.Run("missing refresh config", func(t *testing.T) {
		expectWorkerRequest(t, worker.Request{Op: worker.OpGet, Profile: "alpha"},
			worker.Response{Op: worker.OpGet, OK: true})
		_, _, err := readClusterInWorker(ctx, "alpha")
		require.ErrorContains(t, err, "missing cluster configuration")
	})
	t.Run("process failure", func(t *testing.T) {
		c, err := workerClientInstance()
		require.NoError(t, err)
		saved := c.ExecPath
		t.Cleanup(func() { c.ExecPath = saved })
		c.ExecPath = filepath.Join(t.TempDir(), "missing-worker")
		_, err = runWorkerOperation(ctx, &worker.Request{Op: worker.OpGet})
		require.ErrorContains(t, err, "start worker process")
	})
	t.Run("client initialization failure", func(t *testing.T) {
		saved := workerClientErr
		t.Cleanup(func() { workerClientErr = saved })
		workerClientErr = errors.New("resolve executable failed")
		_, err := runWorkerOperation(ctx, &worker.Request{Op: worker.OpGet})
		require.EqualError(t, err, "resolve executable failed")
	})
}

func TestResourceWorkerDispatch(t *testing.T) {
	for _, op := range []string{worker.OpStart, worker.OpGet, worker.OpAddons, worker.OpDelete} {
		t.Run(op, func(t *testing.T) {
			t.Setenv(worker.EnvInProcess, "")
			d := schema.TestResourceDataRaw(t, GetClusterSchema(), map[string]interface{}{
				"cluster_name": "alpha", "driver": "docker", "addons": []interface{}{"ingress"},
			})
			d.SetId("alpha")
			client := &lib.MinikubeClient{K8sVersion: "v1.30.0"}
			factory := func() (lib.ClusterClient, error) { return client, nil }
			_, err := initialiseMinikubeClient(d, factory)
			require.NoError(t, err)
			cfg := client.GetConfig()
			req := worker.Request{Op: op, Profile: "alpha"}
			resp := worker.Response{Op: op, OK: true}
			switch op {
			case worker.OpStart:
				req.Config = &cfg
				resp.OK, resp.Error = false, "start failed"
			case worker.OpGet:
				resp.ClusterConfig = cfg.ClusterConfig
			case worker.OpAddons:
				req.Enable = []string{"ingress"}
			case worker.OpDelete:
				req.Config = &cfg
			}
			expectWorkerRequest(t, req, resp)
			ctx := context.Background()
			switch op {
			case worker.OpStart:
				diags := resourceClusterCreate(ctx, d, factory)
				require.True(t, diags.HasError())
				require.Contains(t, diags[0].Summary, "start failed")
			case worker.OpGet:
				require.Empty(t, resourceClusterRead(ctx, d, factory))
				require.Equal(t, "2", d.Get("cpus"))
				require.Equal(t, []interface{}{"ingress"}, d.Get("addons").(*schema.Set).List())
			case worker.OpAddons:
				require.Empty(t, resourceClusterUpdate(ctx, d, factory))
				require.Equal(t, []interface{}{"ingress"}, d.Get("addons").(*schema.Set).List())
			case worker.OpDelete:
				require.Empty(t, resourceClusterDelete(ctx, d, factory))
				require.Empty(t, d.Id())
			}
		})
	}
}
