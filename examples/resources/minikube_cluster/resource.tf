provider "minikube" {
  kubernetes_version = "v1.26.3"
}

resource "minikube_cluster" "docker" {
  driver       = "docker"
  cluster_name = "terraform-provider-minikube-acc-docker"
  addons = [
    "default-storageclass",
    "storage-provisioner"
  ]
}

resource "minikube_cluster" "hyperkit" {
  vm           = true
  driver       = "hyperkit"
  cluster_name = "terraform-provider-minikube-acc-hyperkit"
  nodes        = 3
  addons = [
    "dashboard",
    "default-storageclass",
    "ingress",
    "storage-provisioner"
  ]
}

provider "kubernetes" {
  host = minikube_cluster.docker.host

  client_certificate     = minikube_cluster.docker.client_certificate
  client_key             = minikube_cluster.docker.client_key
  cluster_ca_certificate = minikube_cluster.docker.cluster_ca_certificate
}


resource "kubernetes_deployment" "deployment" {
  metadata {
    name = "nginx-example"
    labels = {
      App = "NginxExample"
    }
  }

  spec {
    replicas = 2
    selector {
      match_labels = {
        App = "NginxExample"
      }
    }
    template {
      metadata {
        labels = {
          App = "NginxExample"
        }
      }
      spec {
        container {
          image = "nginx:1.7.8"
          name  = "example"

          port {
            container_port = 80
          }

          resources {
            limits = {
              cpu    = "0.5"
              memory = "512Mi"
            }
            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
      }
    }
  }
}
