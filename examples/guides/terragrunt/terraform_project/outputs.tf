output "client_certificate" {
  value       = minikube_cluster.default.client_certificate
  description = "The cluster client certificate."
  sensitive   = true
}

output "client_key" {
  value       = minikube_cluster.default.client_key
  description = "The cluster client key."
  sensitive   = true
}

output "cluster_ca_certificate" {
  value       = minikube_cluster.default.cluster_ca_certificate
  description = "The cluster CA certificate."
  sensitive   = true
}

output "host" {
  value       = minikube_cluster.default.host
  description = "The cluster host."
}

output "id" {
  value       = minikube_cluster.default.id
  description = "The cluster id."
}
