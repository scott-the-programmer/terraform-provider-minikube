package lib

import "testing"

func TestGetMemoryLimit(t *testing.T) {
	info, err := GetMemoryLimit()
	if err != nil {
		t.Skipf("host memory could not be probed: %v", err)
	}

	if info == nil {
		t.Fatal("GetMemoryLimit() returned a nil MemoryInfo without an error")
	}

	// The reported figure is the host total less 1gb of overhead, so any host
	// large enough to run minikube leaves a positive remainder.
	if info.SystemMemory <= 0 {
		t.Fatalf("GetMemoryLimit() SystemMemory = %d, want > 0", info.SystemMemory)
	}
}
