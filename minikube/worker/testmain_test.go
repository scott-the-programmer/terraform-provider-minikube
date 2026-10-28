package worker

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/minikube/pkg/minikube/config"
)

const testStderrSentinel = "TFMK-TEST-STDERR-SENTINEL"

const defaultTestKubernetesVersion = "v1.30.0"

func TestMain(m *testing.M) {
	if os.Getenv(EnvTestHelper) != "" {
		switch {
		case os.Getenv(EnvTestGarbage) != "":
			os.Stderr.WriteString(testStderrSentinel + " garbage mode\n")
			os.Stdout.WriteString("not a response ~~~ stray output <<<\n")
			os.Exit(0)
		case os.Getenv(EnvTestExit) != "":
			os.Stderr.WriteString(testStderrSentinel + " exit mode\n")
			code, _ := jsonNumber(os.Getenv(EnvTestExit))
			os.Exit(code)
		default:
			os.Exit(Serve(os.Stdin, os.Stdout))
		}
	}
	os.Exit(m.Run())
}

func jsonNumber(raw string) (int, error) {
	var code int
	if err := json.Unmarshal([]byte(raw), &code); err != nil {
		return 3, err
	}
	return code, nil
}

func testClient(t *testing.T, extraEnv ...string) *Client {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test binary: %v", err)
	}
	env := append([]string{EnvTestHelper + "=1"}, extraEnv...)
	return &Client{ExecPath: exe, ExtraEnv: env, StderrTail: 8192}
}

func getProfileRequest(profile string) *Request {
	return &Request{Op: OpGet, Profile: profile}
}

func writeProfileFixture(t *testing.T, home, name string, addons map[string]bool) *config.ClusterConfig {
	t.Helper()

	if addons == nil {
		addons = map[string]bool{"default-storageclass": true, "storage-provisioner": true}
	}
	cc := &config.ClusterConfig{
		Name:          name,
		Driver:        "docker",
		APIServerPort: 8443,
		Nodes: []config.Node{{
			Name:              "",
			Port:              8443,
			KubernetesVersion: defaultTestKubernetesVersion,
			ContainerRuntime:  "docker",
			ControlPlane:      true,
			Worker:            true,
		}},
		KubernetesConfig: config.KubernetesConfig{
			KubernetesVersion: defaultTestKubernetesVersion,
			ClusterName:       "mk-" + name,
			CNI:               "bridge",
			ContainerRuntime:  "docker",
		},
		Addons: addons,
	}

	path := filepath.Join(home, "profiles", name, "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture profile: %v", err)
	}
	b, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture profile: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write fixture profile: %v", err)
	}
	return cc
}

// Minikube appends .minikube to MINIKUBE_HOME.
func isolatedMinikubeHome(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	home := filepath.Join(base, ".minikube")
	t.Setenv("MINIKUBE_HOME", base)
	return home
}

func runServe(t *testing.T, req *Request) *Response {
	t.Helper()

	payload, err := encode(req)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	var out bytes.Buffer
	code := Serve(bytes.NewReader(payload), &out)
	if code != 0 {
		t.Fatalf("Serve exited with code %d", code)
	}

	resp, err := decodeFrame(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func runServeRaw(t *testing.T, req *Request) int {
	t.Helper()
	payload, err := encode(req)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	return Serve(bytes.NewReader(payload), &bytes.Buffer{})
}
