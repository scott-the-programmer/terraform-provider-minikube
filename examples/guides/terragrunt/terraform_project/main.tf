terraform {
  required_providers {
    minikube = {
      source  = "scott-the-programmer/minikube"
      version = "0.3.10"
    }
  }
}

resource "minikube_cluster" "default" {
  cluster_name        = var.cluster_name
  driver              = var.driver
  nodes               = var.nodes
  cpus                = var.cpus
  preload             = true
  cache_images        = true
  auto_update_drivers = true
  install_addons      = true
  addons = [
    "default-storageclass",
    "storage-provisioner"
  ]
}
