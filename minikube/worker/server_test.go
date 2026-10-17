package worker

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"k8s.io/minikube/pkg/minikube/config"
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
