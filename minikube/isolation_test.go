package minikube

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/worker"
)

func TestMain(m *testing.M) {
	if expected := os.Getenv("TF_MINIKUBE_PROVIDER_TEST_REQUEST"); expected != "" {
		var want, got worker.Request
		if err := json.Unmarshal([]byte(expected), &want); err != nil {
			panic(err)
		}
		if err := json.NewDecoder(os.Stdin).Decode(&got); err != nil {
			panic(err)
		}
		if !reflect.DeepEqual(want, got) {
			fmt.Fprintf(os.Stderr, "worker request: got %+v, want %+v", got, want)
			os.Exit(1)
		}
		fmt.Print(worker.ResponseMarker + os.Getenv("TF_MINIKUBE_PROVIDER_TEST_RESPONSE"))
		os.Exit(0)
	}
	os.Setenv("TF_MINIKUBE_IN_PROCESS", "1")
	os.Exit(m.Run())
}
