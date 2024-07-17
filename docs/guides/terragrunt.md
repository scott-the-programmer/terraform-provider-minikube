---
page_title: "terragrunt Guide - terraform-provider-minikube"
subcategory: ""
description: |-
  Guide for creating minikube cluster with terragrunt
---

# Guide: Launching `minikube` clusters with `terragrunt`

`terragrunt` is a tool that wraps `terraform` or `tofu` for reusing modules and generating sources.
This is primarily useful if you're working with a large number of `minikube` configurations, or you're wanting to keep them DRY.

## Prerequisites

* Install [terraform](https://github.com/hashicorp/terraform) or [tofu](https://github.com/opentofu/opentofu)
* Install [terragrunt](https://github.com/gruntwork-io/terragrunt)

## Terragrunt Project with Common Environment

While it's possible to create a working `terragrunt` project with fewer files, this example is the most generally applicable.
The [`examples/terragrunt`](https://github.com/scott-the-programmer/terraform-provider-minikube/tree/main/examples) folder contains two projects: `terraform_project` and `terragrunt_project`.

### terraform_project

The `terraform_project` folder is a valid project and can be initialized using the standard `terraform init`, `terraform plan`, and `terraform apply`.

```shell
cat <<EOF > terraform.tfvars
cluster_name = "minikube"
driver       = "docker"
nodes        = 1
cpus         = 2
memory       = 2048
EOF
terraform init
terraform plan
terraform apply
```

### terragrunt_project

Within `terragrunt_project/dev`, there are two runnable configurations named `cluster-a` and `cluster-b` that reuse the `terraform_project` with shared inputs; they're run from their working directories using `terragrunt init`, `terragrunt plan`, and `terragrunt apply`.

```shell
cat <<EOF > terraform.tfvars
nodes  = 1
cpus   = 2
memory = 2048
EOF
terragrunt init
terragrunt plan
terragrunt apply
```
