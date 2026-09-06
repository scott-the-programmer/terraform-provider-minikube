cluster_name = "tf-minikube-e2e-podman"
driver       = "podman"
vm           = false
nodes        = 1
cpus         = "2"
memory       = "4g"

# minikube's podman driver does not support the docker runtime; cri-o is the
# combination minikube itself exercises.
container_runtime = "cri-o"
addons            = ["default-storageclass", "storage-provisioner"]
