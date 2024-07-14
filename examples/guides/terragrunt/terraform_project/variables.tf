variable "cluster_name" {
  type        = string
  description = "The minikube name."
}

variable "driver" {
  type        = string
  description = "The minikube driver."
}

variable "nodes" {
  type        = number
  description = "The number of nodes."
}

variable "cpus" {
  type        = number
  description = "The number of cpus."
}

variable "memory" {
  type        = number
  description = "The amount of memory."
}
