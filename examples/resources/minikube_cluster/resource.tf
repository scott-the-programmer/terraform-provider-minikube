provider "minikube" { 
  kubernetes_version = "v1.23.3"
}

resource "minikube_cluster" "docker" {
  driver = "docker"
  cluster_name = "terraform-provider-minikube-acc-docker"
  addons = [
    "default-storageclass",
  ]
}

resource "minikube_cluster" "hyperkit" {
  vm = true
  driver = "hyperkit"
  cluster_name = "terraform-provider-minikube-acc-hyperkit"
  nodes = 3
  addons = [
    "dashboard",
    "default-storageclass",
    "ingress"
  ]
}
