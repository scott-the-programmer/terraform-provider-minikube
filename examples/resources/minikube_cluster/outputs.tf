output "docker_host" {
  value = minikube_cluster.docker.host
}

output "docker_key" {
  sensitive = true
  value = minikube_cluster.docker.client_key
}

output "docker_certificate" {
  sensitive = true
  value = minikube_cluster.docker.client_certificate
}

output "docker_ca" {
  sensitive = true
  value = minikube_cluster.docker.cluster_ca_certificate
}
