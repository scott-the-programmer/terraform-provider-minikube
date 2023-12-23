//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package lib

import "k8s.io/minikube/pkg/minikube/download"

type Downloader interface {
	ISO(urls []string, skipChecksum bool) (string, error)
	PreloadTarball(k8sVersion, containerRuntime, driver string) error
}

type MinikubeDownloader struct {
}

func NewMinikubeDownloader() *MinikubeDownloader {
	return &MinikubeDownloader{}
}

func (m *MinikubeDownloader) ISO(urls []string, skipChecksum bool) (string, error) {
	return download.ISO(urls, skipChecksum)
}

func (m *MinikubeDownloader) PreloadTarball(k8sVersion, containerRuntime, driver string) error {
	return download.Preload(k8sVersion, containerRuntime, driver)
}
