package worker

import (
	"bytes"
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"k8s.io/minikube/pkg/minikube/config"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
)

func TestRequestResponseRoundTrip(t *testing.T) {
	start := &Request{
		Op:      OpStart,
		Profile: "alpha",
		Config: &lib.MinikubeClientConfig{
			ClusterName: "alpha",
			ClusterConfig: &config.ClusterConfig{
				Name:       "alpha",
				Driver:     "docker",
				DNSServers: []netip.Addr{mustAddr(t, "10.0.0.5")},
				KubernetesConfig: config.KubernetesConfig{
					KubernetesVersion: "v1.30.0",
				},
			},
		},
	}

	payload, err := encode(start)
	require.NoError(t, err)

	var decoded Request
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, OpStart, decoded.Op)
	require.Equal(t, "alpha", decoded.Profile)
	require.NotNil(t, decoded.Config.ClusterConfig)
	require.Equal(t, "docker", decoded.Config.ClusterConfig.Driver)
	require.NotNil(t, decoded.Config.ClusterConfig.DNSServers)
	require.Equal(t, "10.0.0.5", decoded.Config.ClusterConfig.DNSServers[0].String())
}

func TestDecodeFrame(t *testing.T) {
	resp := &Response{Op: OpGet, OK: true, ClusterConfig: &config.ClusterConfig{Name: "bravo"}}
	var buf bytes.Buffer
	require.NoError(t, writeFrame(&buf, resp))

	stray := []byte("some stray framework output\n" + buf.String() + "\n")
	got, err := decodeFrame(stray)
	require.NoError(t, err)
	require.Equal(t, OpGet, got.Op)
	require.True(t, got.OK)
	require.Equal(t, "bravo", got.ClusterConfig.Name)

	_, err = decodeFrame([]byte("no marker here"))
	require.Error(t, err)

	_, err = decodeFrame(nil)
	require.Error(t, err)

	_, err = decodeFrame([]byte(ResponseMarker + "{"))
	require.Error(t, err)
}

func TestKubeconfigOutputsRoundTrip(t *testing.T) {
	out := KubeconfigOutputs{
		ClusterName:          "charlie",
		ClusterServerAddress: "https://127.0.0.1:8443",
		ClientKey:            "/tmp/k/charlie/key",
		ClientCertificate:    "/tmp/k/charlie/cert",
		CertificateAuthority: "/tmp/k/charlie/ca",
	}
	var buf bytes.Buffer
	require.NoError(t, writeFrame(&buf, &Response{Op: OpStart, OK: true, Kubeconfig: &out}))
	got, err := decodeFrame(buf.Bytes())
	require.NoError(t, err)
	require.Equal(t, out, *got.Kubeconfig)

	settings := got.Kubeconfig.Settings()
	require.Equal(t, out.ClusterName, settings.ClusterName)
	require.Equal(t, out.ClusterServerAddress, settings.ClusterServerAddress)
	require.Equal(t, out.ClientKey, settings.ClientKey)
	require.Equal(t, out.ClientCertificate, settings.ClientCertificate)
	require.Equal(t, out.CertificateAuthority, settings.CertificateAuthority)
}

func TestDeadlinePropagation(t *testing.T) {
	deadline := time.Now().Add(3 * time.Minute).Round(time.Second)
	req := &Request{Op: OpGet, Profile: "zulu", Deadline: deadline}
	payload, err := encode(req)
	require.NoError(t, err)

	var decoded Request
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, deadline.Unix(), decoded.Deadline.Unix(), "deadline should survive encoding")
}

func mustAddr(t *testing.T, raw string) netip.Addr {
	t.Helper()
	a, err := netip.ParseAddr(raw)
	require.NoError(t, err)
	return a
}
