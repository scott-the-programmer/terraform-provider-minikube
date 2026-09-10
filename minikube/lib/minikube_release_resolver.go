package lib

import (
	"fmt"
	"runtime"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/version"
)

var (
	releaseBase = fmt.Sprintf("https://github.com/kubernetes/minikube/releases/download/%s",
		version.ISOVersion)
)

func GetMinikubeIso() string {
	return fmt.Sprintf("%s/minikube-%s-%s.iso", releaseBase, version.ISOVersion, runtime.GOARCH)
}
