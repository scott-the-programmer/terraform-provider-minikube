package service

import (
	"fmt"
	"runtime"
	"terraform-provider-minikube/m/v2/minikube/version"
)

var (
	releaseBase = fmt.Sprintf("https://github.com/kubernetes/minikube/releases/download/%s",
		version.Version)
)

func GetMinikubeIso() string {
	return fmt.Sprintf("%s/minikube-%s-%s.iso", releaseBase, version.Version, runtime.GOARCH)
}
