// Package worker isolates Minikube globals in per-operation processes.
package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
)

const (
	EnvWorkerChild = "TF_MINIKUBE_WORKER"

	// EnvInProcess runs operations serially in the provider.
	EnvInProcess = "TF_MINIKUBE_IN_PROCESS"
)

const (
	EnvTestHelper  = "TF_MINIKUBE_WORKER_TEST_HELPER"
	EnvTestDelayMs = "TF_MINIKUBE_WORKER_TEST_DELAY_MS"
	EnvTestGarbage = "TF_MINIKUBE_WORKER_TEST_GARBAGE"
	EnvTestExit    = "TF_MINIKUBE_WORKER_TEST_EXIT"
)

const OpInvalid = "invalid-request"

const (
	OpStart  = "start"
	OpAddons = "addons"
	OpGet    = "get"
	OpDelete = "delete"
	OpProbe  = "probe-isolation"
)

// ResponseMarker delimits JSON from stray stdout.
const ResponseMarker = "tf-minikube-worker-response-v1:"

type Request struct {
	Op       string    `json:"op"`
	Profile  string    `json:"profile"`
	Deadline time.Time `json:"deadline,omitempty"`
	// Config is required for start and optional for delete.
	Config  *lib.MinikubeClientConfig `json:"config,omitempty"`
	Enable  []string                  `json:"enable,omitempty"`
	Disable []string                  `json:"disable,omitempty"`
	Probe   string                    `json:"probe,omitempty"`
}

type KubeconfigOutputs struct {
	ClusterName          string `json:"cluster_name,omitempty"`
	ClusterServerAddress string `json:"cluster_server_address,omitempty"`
	ClientKey            string `json:"client_key,omitempty"`
	ClientCertificate    string `json:"client_certificate,omitempty"`
	CertificateAuthority string `json:"certificate_authority,omitempty"`
}

type Response struct {
	Op            string                `json:"op"`
	OK            bool                  `json:"ok"`
	Error         string                `json:"error,omitempty"`
	Kubeconfig    *KubeconfigOutputs    `json:"kubeconfig,omitempty"`
	ClusterConfig *config.ClusterConfig `json:"cluster_config,omitempty"`
	Probe         string                `json:"probe,omitempty"`
}

func (k *KubeconfigOutputs) Settings() *kubeconfig.Settings {
	return &kubeconfig.Settings{
		ClusterName:          k.ClusterName,
		ClusterServerAddress: k.ClusterServerAddress,
		ClientKey:            k.ClientKey,
		ClientCertificate:    k.ClientCertificate,
		CertificateAuthority: k.CertificateAuthority,
	}
}

func encode(r *Request) ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode worker request: %w", err)
	}
	return b, nil
}

func decodeFrame(raw []byte) (*Response, error) {
	marker := len(raw)
	for i := range raw {
		if i+len(ResponseMarker) <= len(raw) && string(raw[i:i+len(ResponseMarker)]) == ResponseMarker {
			marker = i + len(ResponseMarker)
			break
		}
	}
	if marker >= len(raw) {
		return nil, fmt.Errorf("worker wrote no response frame")
	}
	var resp Response
	if err := json.Unmarshal(raw[marker:], &resp); err != nil {
		return nil, fmt.Errorf("decoding worker response: %w", err)
	}
	if resp.Op == "" {
		return nil, fmt.Errorf("worker response missing operation")
	}
	return &resp, nil
}

func writeFrame(w interface{ Write([]byte) (int, error) }, resp *Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode worker response: %w", err)
	}
	doc := []byte("\n" + ResponseMarker + string(b) + "\n")
	_, err = w.Write(doc)
	return err
}
