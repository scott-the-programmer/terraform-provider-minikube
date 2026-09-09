cluster_name      = "tf-minikube-e2e-docker"
driver            = "docker"
vm                = false
nodes             = 1
cpus              = "2"
memory            = "4g"
container_runtime = "docker"
addons            = ["default-storageclass", "storage-provisioner"]
