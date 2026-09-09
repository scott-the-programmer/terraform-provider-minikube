variable "cluster_name" {
  description = "Name of the minikube profile to create."
  type        = string
}

variable "driver" {
  description = "minikube driver to use (docker, qemu2, podman, ...)."
  type        = string
}

variable "vm" {
  description = "Restrict minikube to VM drivers only. Required by qemu2."
  type        = bool
  default     = false
}

variable "nodes" {
  description = "Number of nodes in the cluster."
  type        = number
  default     = 1
}

variable "cpus" {
  description = "CPUs allocated to the cluster."
  type        = string
  default     = "2"
}

variable "memory" {
  description = "Memory allocated to the cluster."
  type        = string
  default     = "4g"
}

variable "kubernetes_version" {
  description = "Kubernetes version to run. Empty means the minikube default."
  type        = string
  default     = ""
}

variable "container_runtime" {
  description = "Container runtime to run inside the cluster."
  type        = string
  default     = "docker"
}

variable "addons" {
  description = "Addons to enable on the cluster."
  type        = list(string)
  default     = ["default-storageclass", "storage-provisioner"]
}
