# terraform-provider-minikube

 [![Go Report Card](https://goreportcard.com/badge/github.com/scott-the-programmer/terraform-provider-minikube)](https://goreportcard.com/report/github.com/scott-the-programmer/terraform-provider-minikube)
<a href="https://codeclimate.com/github/scott-the-programmer/terraform-provider-minikube/maintainability"><img src="https://api.codeclimate.com/v1/badges/dd45aac40e7019502245/maintainability" /></a>
<a href="https://codeclimate.com/github/scott-the-programmer/terraform-provider-minikube/test_coverage"><img src="https://api.codeclimate.com/v1/badges/dd45aac40e7019502245/test_coverage" /></a>

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
  kubernetes_version = "v1.26.3"
}

resource "minikube_cluster" "cluster" {
  vm = true
  driver = "hyperkit"
  addons = [
    "dashboard",
    "default-storageclass",
    "ingress"
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

## Want to help out?

See [the contributing doc](./contributing.md) if you wish to get into the details of this terraform minikube provider!
