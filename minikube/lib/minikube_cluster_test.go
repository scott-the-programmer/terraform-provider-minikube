package lib

import (
	"bytes"
	"io"
	"testing"

	"k8s.io/minikube/pkg/minikube/assets"
)

func TestResetAddonAssets(t *testing.T) {
	asset := assets.Addons["dashboard"].Assets[0]
	want, err := asset.ReadFile(asset.GetSourcePath())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := io.Copy(io.Discard, asset); err != nil {
			t.Fatal(err)
		}
		if err := resetAddonAssets(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(asset)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("installation %d: addon contents were not restored", i+1)
		}
	}
	t.Cleanup(func() {
		if err := resetAddonAssets(); err != nil {
			t.Error(err)
		}
	})
}

func TestNewMinikubeClusterInitializesCommandOptions(t *testing.T) {
	cluster := NewMinikubeCluster()

	if cluster.commandOptions == nil {
		t.Fatal("NewMinikubeCluster() commandOptions is nil")
	}
}
