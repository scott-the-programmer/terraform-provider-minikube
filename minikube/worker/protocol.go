// Package worker isolates embedded Minikube operations from the Terraform
// plugin process by re-executing the provider binary in a dedicated worker
// process for every lifecycle operation.
//
// Minikube's pkg/minikube/node start and provision paths persist cluster
// configuration through process-global state (Viper singletons), so running
// two cluster operations in one process can save cluster A's configuration
// under cluster B's profile. Each worker starts fresh, sets the operation's
// own globals, does one job, and exits, so settings cannot leak between
// operations. Terraform state handling stays in the parent: the parent
// reconstructs the operation's full configuration into every request, and
// workers never write Terraform state.
//
// Child processes are selected before Terraform plugin serving (see main.go)
// using the EnvWorkerChild variable. Parents opt back into single-process
// execution by setting EnvInProcess. Responses ride on stdout behind a magic
// marker so stray framework or upstream writes cannot corrupt framing;
// worker logs go to stderr.
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
	// EnvWorkerChild selects worker mode in a re-executed provider process.
	// Set only by the parent executor.
	EnvWorkerChild = "TF_MINIKUBE_WORKER"

	// EnvInProcess opts the parent out of worker isolation so operations run
	// in the plugin process again. Safety reverts to the broad creation
	// mutex, which serializes all cluster operations, so only set this when
	// the host cannot support concurrent workers.
	EnvInProcess = "TF_MINIKUBE_IN_PROCESS"
)

// Test-only seams, honored only inside worker processes. They do not change
// production behavior.
const (
	// EnvTestHelper marks the test binary re-execution used by worker tests.
	EnvTestHelper = "TF_MINIKUBE_WORKER_TEST_HELPER"
	// EnvTestDelayMs delays the worker's response so parents can exercise
	// cancellation and reaping against a live child.
	EnvTestDelayMs = "TF_MINIKUBE_WORKER_TEST_DELAY_MS"
	// EnvTestGarbage makes a worker emit a malformed response. The sentry value
	// is also written to stderr to verify diagnostic tails.
	EnvTestGarbage = "TF_MINIKUBE_WORKER_TEST_GARBAGE"
	// EnvTestExit makes a worker exit with the given nonzero code without
	// responding, simulating upstream termination such as exit.Error.
	EnvTestExit = "TF_MINIKUBE_WORKER_TEST_EXIT"
)

// OpInvalid names responses for requests that could not be decoded.
const OpInvalid = "invalid-request"

// Supported request operations.
const (
	// OpStart creates or reuses a cluster (downloads, provisioning, startup,
	// additional nodes, addons) exactly like MinikubeClient.Start.
	OpStart = "start"
	// OpAddons toggles addon sets on an existing profile.
	OpAddons = "addons"
	// OpGet loads the saved profile configuration for state refresh.
	OpGet = "get"
	// OpDelete destroys a profile and its machine directories.
	OpDelete = "delete"
	// OpProbe reports which operation-global profile value the worker observed,
	// proving configuration cannot leak across the process boundary.
	OpProbe = "probe-isolation"
)

// ResponseMarker guards the response framing on stdout. Everything between the
// marker and the end of stdout is one JSON document; any pre-marker output
// (framework writing to stdout by accident, upstream logging, etc.) is ignored.
const ResponseMarker = "tf-minikube-worker-response-v1:"

// Request describes one isolated operation. Config carries the complete
// reconstructed configuration so every fresh worker can run without observing
// any other operation's globals.
type Request struct {
	// Op selects the operation.
	Op string `json:"op"`
	// Profile is the canonical Minikube profile the operation acts on.
	Profile string `json:"profile"`
	// Deadline is a wall-clock cutoff propagated to the child. Parent context
	// cancellation still applies directly.
	Deadline time.Time `json:"deadline,omitempty"`
	// Config holds the full client configuration for start and delete.
	Config *lib.MinikubeClientConfig `json:"config,omitempty"`
	// Enable lists addons to switch on for OpAddons.
	Enable []string `json:"enable,omitempty"`
	// Disable lists addons to switch off for OpAddons.
	Disable []string `json:"disable,omitempty"`
	// Probe carries the profile value to set for OpProbe.
	Probe string `json:"probe,omitempty"`
}

// KubeconfigOutputs is the serializable subset of kubeconfig.Settings the
// provider reads after creation.
type KubeconfigOutputs struct {
	// ClusterName is the kubeconfig context created for the cluster.
	ClusterName string `json:"cluster_name,omitempty"`
	// ClusterServerAddress is the API server endpoint.
	ClusterServerAddress string `json:"cluster_server_address,omitempty"`
	// ClientKey is the path to the client key file.
	ClientKey string `json:"client_key,omitempty"`
	// ClientCertificate is the path to the client certificate file.
	ClientCertificate string `json:"client_certificate,omitempty"`
	// CertificateAuthority is the path to the cluster CA file.
	CertificateAuthority string `json:"certificate_authority,omitempty"`
}

// Response reports the outcome of a single operation. OK is false when the
// operation failed inside the worker; Error then carries the failure message
// alongside the stderr tail the parent attaches. A missing or malformed
// response, or a nonzero child exit, surfaces as a parent error instead and
// may hint at cancellation, a worker crash, or an upstream os.Exit.
type Response struct {
	// Op echoes the request operation for traceability.
	Op string `json:"op"`
	// OK records whether the operation was completed.
	OK bool `json:"ok"`
	// Error describes the operation failure when OK is false.
	Error string `json:"error,omitempty"`
	// Kubeconfig carries creation outputs for OpStart.
	Kubeconfig *KubeconfigOutputs `json:"kubeconfig,omitempty"`
	// ClusterConfig carries the saved profile for OpGet.
	ClusterConfig *config.ClusterConfig `json:"cluster_config,omitempty"`
	// Probe reports the profile global observed inside the worker for OpProbe.
	Probe string `json:"probe,omitempty"`
}

// Settings rebuilds the kubeconfig.Settings the provider consumes after a
// start, from the fields that safely cross the process boundary.
func (k *KubeconfigOutputs) Settings() *kubeconfig.Settings {
	return &kubeconfig.Settings{
		ClusterName:          k.ClusterName,
		ClusterServerAddress: k.ClusterServerAddress,
		ClientKey:            k.ClientKey,
		ClientCertificate:    k.ClientCertificate,
		CertificateAuthority: k.CertificateAuthority,
	}
}

// encode renders a request as JSON for the child's stdin pipe.
func encode(r *Request) ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode worker request: %w", err)
	}
	return b, nil
}

// decodeFrame extracts a response document placed behind ResponseMarker. Any
// bytes preceding the marker are ignored.
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

// writeFrame puts a response on the output writer behind the marker.
func writeFrame(w interface{ Write([]byte) (int, error) }, resp *Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode worker response: %w", err)
	}
	doc := []byte("\n" + ResponseMarker + string(b) + "\n")
	_, err = w.Write(doc)
	return err
}
