package lib

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/minikube/pkg/minikube/assets"
	"k8s.io/minikube/pkg/minikube/localpath"
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

func TestRmdir(t *testing.T) {
	root := t.TempDir()

	// A missing directory is not an error - Delete() calls rmdir unconditionally.
	missing := filepath.Join(root, "missing")
	if err := rmdir(missing); err != nil {
		t.Fatalf("rmdir(%q) error = %v, want nil", missing, err)
	}

	populated := filepath.Join(root, "machines", "cluster")
	if err := os.MkdirAll(populated, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(populated, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := rmdir(populated); err != nil {
		t.Fatalf("rmdir(%q) error = %v, want nil", populated, err)
	}
	if _, err := os.Stat(populated); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not-exist", populated, err)
	}
}

func TestMakeAllMinikubeDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv(localpath.MinikubeHome, home)

	makeAllMinikubeDirectories()

	for _, dir := range []string{"certs", "machines", "cache", "config", "addons", "files", "logs"} {
		path := localpath.MakeMiniPath(dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("os.Stat(%q) error = %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", path)
		}
	}

	// Existing directories are left alone rather than treated as a failure.
	makeAllMinikubeDirectories()
}
