# terraform-provider-minikube

 [![Go Report Card](https://goreportcard.com/badge/github.com/scott-the-programmer/terraform-provider-minikube)](https://goreportcard.com/report/github.com/scott-the-programmer/terraform-provider-minikube)
[![codecov](https://codecov.io/gh/scott-the-programmer/terraform-provider-minikube/graph/badge.svg?token=MH35FEWVAH)](https://codecov.io/gh/scott-the-programmer/terraform-provider-minikube)

A terraform provider for [minikube!](https://minikube.sigs.k8s.io/docs/)

The goal of this project is to allow developers to create minikube clusters and integrate it with common kubernetes terraform providers such as [hashicorp/kubernetes](https://registry.terraform.io/providers/hashicorp/kubernetes/2.12.1) and [hashicorp/helm](https://registry.terraform.io/providers/hashicorp/helm/2.6.0) all within the comfort of Minikube!

You can learn more about how to use the provider at https://registry.terraform.io/providers/scott-the-programmer/minikube/latest/docs

## Installing your preferred driver

```bash
minikube start --vm=true --driver=hyperkit --download-only
minikube start --vm=true --driver=hyperv --download-only
minikube start --driver=docker --download-only
```

Some drivers require a bit of prerequisite setup, so it's best to visit [https://minikube.sigs.k8s.io/docs/drivers/](https://minikube.sigs.k8s.io/docs/drivers/) first

## Usage

```terraform
provider minikube {
  kubernetes_version = "v1.30.0"
}

resource "minikube_cluster" "cluster" {
  vm      = true
  driver  = "hyperkit"
  cni     = "bridge"
  addons  = [
    "dashboard",
    "default-storageclass",
    "ingress",
    "storage-provisioner"
  ]
}
```

You can use `minikube` to verify the cluster is up & running

```console
> minikube profile list

|----------------------------------------|-----------|---------|---------------|------|---------|---------|-------|
|                Profile                 | VM Driver | Runtime |      IP       | Port | Version | Status  | Nodes |
|----------------------------------------|-----------|---------|---------------|------|---------|---------|-------|
| terraform-provider-minikube            | hyperkit  | docker  | 192.168.64.42 | 8443 | v1.26.3 | Running |     1 |
|----------------------------------------|-----------|---------|---------------|------|---------|---------|-------|
```

## Outputs

In order to integrate the minikube providers with other k8s providers, you can reference the following outputs

- `client_certificate` (string, sensitive) client certificate used in cluster
- `client_key` (string, sensitive) client key for cluster
- `cluster_ca_certificate` (string, sensitive) certificate authority for cluster
- `host` (string) the host name for the cluster

These outputs are consistent across supported by all minikube cluster types

i.e.

```terraform
provider "kubernetes" {
  host = minikube_cluster.cluster.host

  client_certificate     = minikube_cluster.cluster.client_certificate
  client_key             = minikube_cluster.cluster.client_key
  cluster_ca_certificate = minikube_cluster.cluster.cluster_ca_certificate
}
```

## Want to help out?

See [the contributing doc](./contributing.md) if you wish to get into the details of this terraform minikube provider!
