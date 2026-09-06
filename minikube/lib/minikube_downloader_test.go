package lib

import "testing"

func TestNewMinikubeDownloader(t *testing.T) {
	d := NewMinikubeDownloader()
	if d == nil {
		t.Fatal("NewMinikubeDownloader() = nil")
	}

	// ISO/PreloadTarball reach out to the network, so only the wiring is checked here.
	var _ Downloader = d
}
