# terraform-provider-minikube

<a href="https://codeclimate.com/github/scott-the-programmer/terraform-provider-minikube/maintainability"><img src="https://api.codeclimate.com/v1/badges/dd45aac40e7019502245/maintainability" /></a>

A terraform provider for [minikube!](https://minikube.sigs.k8s.io/docs/)

The goal of this project is to allow developers to create minikube clusters and integrate it with common kubernetes terraform providers such as [hashicorp/kubernetes](https://registry.terraform.io/providers/hashicorp/kubernetes/2.12.1) and [hashicorp/helm](https://registry.terraform.io/providers/hashicorp/helm/2.6.0) all within the comfort of Minikube!

You can learn more about how to use the provider at https://registry.terraform.io/providers/scott-the-programmer/minikube/latest/docs

## Installing your preferred driver

If you don't have minikube installed, or have never run minikube before, you'll need to install your corresponding driver first

This requires _admin_ permissions

### Minikube

```bash
minikube --vm=true --driver=hyperkit --download-only
minikube --vm=true --driver=hyperv --download-only
minikube --driver=docker --download-only
```

### Manual

You can find the drivers published in the [minikube releases section](https://github.com/kubernetes/minikube/releases). Simply download the 
preferred driver and copy it to your .minikube/bin folder and ensure the current user has sufficient access

### Living dangerously (discouraged)

```bash
#x86_64
curl https://raw.githubusercontent.com/scott-the-programmer/terraform-provider-minikube/main/bootstrap/install-driver.sh | sudo bash -s "kvm2"

#arm64
curl https://raw.githubusercontent.com/scott-the-programmer/terraform-provider-minikube/main/bootstrap/install-driver.sh | sudo bash -s "kvm2" "arm64"
```

## Usage

```terraform
provider minikube {
  kubernetes_version = "v1.23.3"
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
| terraform-provider-minikube            | hyperkit  | docker  | 192.168.64.42 | 8443 | v1.23.3 | Running |     1 |
|----------------------------------------|-----------|---------|---------------|------|---------|---------|-------|
```

## Want to help out?

See [the contributing](./contributing.md) if you wish to get into the details of this terraform minikube provider!