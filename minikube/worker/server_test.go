package worker

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
)

func TestServeGetOp(t *testing.T) {
	home := isolatedMinikubeHome(t)
	writeProfileFixture(t, home, "delta", nil)

	resp := runServe(t, &Request{Op: OpGet, Profile: "delta"})

	require.True(t, resp.OK, "get should succeed: %s", resp.Error)
	require.Equal(t, OpGet, resp.Op)
	require.NotNil(t, resp.ClusterConfig)
	require.Equal(t, "delta", resp.ClusterConfig.Name)
	require.Equal(t, defaultTestKubernetesVersion, resp.ClusterConfig.KubernetesConfig.KubernetesVersion)
	require.True(t, resp.ClusterConfig.Addons["default-storageclass"])
}

func TestServeGetOpMissingProfile(t *testing.T) {
	isolatedMinikubeHome(t)

	resp := runServe(t, &Request{Op: OpGet, Profile: "ghost"})

	require.False(t, resp.OK)
	require.Equal(t, OpGet, resp.Op)
	require.Contains(t, resp.Error, "ghost")
	require.Contains(t, resp.Error, "not found")
}

func TestServeProbeOpReportsOwnProfile(t *testing.T) {
	resp := runServe(t, &Request{Op: OpProbe, Probe: "probe-in-process"})

	require.True(t, resp.OK, "probe should succeed: %s", resp.Error)
	require.Equal(t, "probe-in-process", resp.Probe)

	t.Cleanup(func() {
		// The probe mutates the process-global Viper profile; don't let it
		// leak into sibling tests of this process.
		resetTestViperProfile()
	})
}

func TestServeUnknownOperation(t *testing.T) {
	resp := runServe(t, &Request{Op: "teleport", Profile: "echo"})

	require.False(t, resp.OK)
	require.Contains(t, resp.Error, "teleport")
	require.Contains(t, resp.Error, "unknown operation")
}

func TestServeMalformedRequest(t *testing.T) {
	var out bytes.Buffer
	code := Serve(bytes.NewReader([]byte("{invalid")), &out)
	require.Equal(t, 1, code)

	resp, err := decodeFrame(out.Bytes())
	require.NoError(t, err)
	require.False(t, resp.OK)
	require.Contains(t, resp.Error, "decoding worker request")
}

func TestServeAddonsOpRewritesProfile(t *testing.T) {
	home := isolatedMinikubeHome(t)
	writeProfileFixture(t, home, "hotel", map[string]bool{"default-storageclass": true, "storage-provisioner": true})

	// These storage-class addons take a fast "cluster not running" path in
	// upstream's callbacks, so the test is hermetic; addons like ingress would
	// try to reach a real node.
	resp := runServe(t, &Request{
		Op:      OpAddons,
		Profile: "hotel",
		Enable:  []string{"storage-provisioner-rancher"},
	})
	require.True(t, resp.OK, "addons should apply: %s", resp.Error)

	resp = runServe(t, &Request{
		Op:      OpAddons,
		Profile: "hotel",
		Disable: []string{"default-storageclass", "storage-provisioner-rancher"},
	})
	require.True(t, resp.OK, "addons should apply: %s", resp.Error)

	// Verify the saved profile actually changed in the disposable home.
	reloaded, err := config.Load("hotel")
	require.NoError(t, err)
	require.False(t, reloaded.Addons["default-storageclass"], "default-storageclass should be saved disabled")
	require.False(t, reloaded.Addons["storage-provisioner-rancher"], "rancher should be saved disabled")
	require.True(t, reloaded.Addons["storage-provisioner"], "unmentioned addons must stay untouched")
}

func resetTestViperProfile() {
	viper.Set(config.ProfileName, "")
}

func TestWorkerConfiguration(t *testing.T) {
	cfg := lib.MinikubeClientConfig{
		ClusterName: "alpha", ClusterConfig: &config.ClusterConfig{Name: "alpha"},
		Nodes: 3, HA: true, NativeSsh: true, DeleteOnFailure: true,
		Addons: []string{"ingress"}, IsoUrls: []string{"file:///minikube.iso"},
	}
	req := Request{Profile: "alpha", Config: &cfg}
	require.Equal(t, cfg, startClient(req).GetConfig())
	require.Equal(t, cfg.ClusterConfig, deleteClient(req).GetConfig().ClusterConfig)
	require.Equal(t, "alpha", deleteClient(Request{Profile: "alpha"}).GetConfig().ClusterName)
	kc := &kubeconfig.Settings{
		ClusterName: "alpha", ClusterServerAddress: "https://127.0.0.1:8443",
		ClientKey: "/key", ClientCertificate: "/cert", CertificateAuthority: "/ca",
	}
	require.Equal(t, kc, outputsFromKubeconfig(kc).Settings())
	deadline := time.Now().Add(time.Minute)
	ctx, cancel := opContext(deadline)
	defer cancel()
	got, ok := ctx.Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, got)
}

func TestServeCorruptProfile(t *testing.T) {
	home := isolatedMinikubeHome(t)
	writeProfileFixture(t, home, "corrupt", nil)
	require.NoError(t, os.WriteFile(filepath.Join(home, "profiles", "corrupt", "config.json"), []byte("{"), 0o600))
	resp := runServe(t, &Request{Op: OpGet, Profile: "corrupt"})
	require.False(t, resp.OK)
	require.Contains(t, resp.Error, "load profile")
	resp = runServe(t, &Request{Op: OpDelete, Profile: "corrupt"})
	require.False(t, resp.OK)
	require.Contains(t, resp.Error, "load cluster for deletion")
}

func TestServeAddonErrors(t *testing.T) {
	isolatedMinikubeHome(t)
	for _, req := range []Request{
		{Op: OpAddons, Profile: "missing", Enable: []string{"unknown-addon"}},
		{Op: OpAddons, Profile: "missing", Disable: []string{"unknown-addon"}},
	} {
		resp := runServe(t, &req)
		require.False(t, resp.OK)
		require.Contains(t, resp.Error, "unknown-addon")
	}
}

type brokenStream struct{}

func (brokenStream) Read([]byte) (int, error)  { return 0, errors.New("read failed") }
func (brokenStream) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestServeIOErrors(t *testing.T) {
	var out bytes.Buffer
	require.Equal(t, 1, Serve(brokenStream{}, &out))
	resp, err := decodeFrame(out.Bytes())
	require.NoError(t, err)
	require.Contains(t, resp.Error, "read failed")
	require.Equal(t, 1, Serve(bytes.NewBufferString(`{"op":"teleport"}`), brokenStream{}))
}

func TestServeInvalidDelay(t *testing.T) {
	t.Setenv(EnvTestDelayMs, "invalid")
	resp := runServe(t, &Request{Op: OpGet})
	require.False(t, resp.OK)
	require.Contains(t, resp.Error, EnvTestDelayMs)
	t.Setenv(EnvTestDelayMs, "0")
	require.NoError(t, applyTestSeams())
}

func TestWorkerIsolationMode(t *testing.T) {
	t.Setenv(EnvWorkerChild, "")
	t.Setenv(EnvInProcess, "")
	require.True(t, IsolationEnabled())
	t.Setenv(EnvInProcess, "1")
	require.False(t, IsolationEnabled())
	t.Setenv(EnvInProcess, "")
	t.Setenv(EnvWorkerChild, "1")
	require.True(t, IsWorkerChild())
	require.False(t, IsolationEnabled())
}
