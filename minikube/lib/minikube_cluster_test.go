package lib

import "testing"

func TestNewMinikubeClusterInitializesCommandOptions(t *testing.T) {
	cluster := NewMinikubeCluster()

	if cluster.commandOptions == nil {
		t.Fatal("NewMinikubeCluster() commandOptions is nil")
	}
}
