package minikube

import (
	"os"
	"testing"
)

// TestMain keeps the provider unit tests on the in-process execution path.
// Worker isolation re-executes the provider binary, and a go test binary has
// no worker entry point, so the mock-based suites must exercise the documented
// fallback instead.
func TestMain(m *testing.M) {
	os.Setenv("TF_MINIKUBE_IN_PROCESS", "1")
	os.Exit(m.Run())
}
