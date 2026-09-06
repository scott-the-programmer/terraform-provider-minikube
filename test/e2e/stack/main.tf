# A single, driver-agnostic stack. Every flavour under ../flavours feeds this
# same configuration a different set of variables, so the assertions that run
# afterwards are identical no matter which driver is under test.
resource "minikube_cluster" "this" {
  cluster_name       = var.cluster_name
  driver             = var.driver
  vm                 = var.vm
  nodes              = var.nodes
  cpus               = var.cpus
  memory             = var.memory
  kubernetes_version = var.kubernetes_version
  container_runtime  = var.container_runtime
  addons             = var.addons

  # A half-started cluster left on disk wedges every subsequent run.
  delete_on_failure = true
}
