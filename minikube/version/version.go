package version

const (
	Version = "v1.38.1" //	matches k8s.io/minikube v1.38.1
	// ISOVersion matches ISO_VERSION in the upstream Minikube Makefile.
	// Patch releases can reuse the ISO from an earlier release.
	ISOVersion = "v1.38.0"
)
