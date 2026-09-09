output "cluster_name" {
  value = minikube_cluster.this.cluster_name
}

output "driver" {
  value = minikube_cluster.this.driver
}

output "nodes" {
  value = minikube_cluster.this.nodes
}

output "host" {
  value = minikube_cluster.this.host
}

output "client_certificate" {
  sensitive = true
  value     = minikube_cluster.this.client_certificate
}

output "client_key" {
  sensitive = true
  value     = minikube_cluster.this.client_key
}

output "cluster_ca_certificate" {
  sensitive = true
  value     = minikube_cluster.this.cluster_ca_certificate
}
