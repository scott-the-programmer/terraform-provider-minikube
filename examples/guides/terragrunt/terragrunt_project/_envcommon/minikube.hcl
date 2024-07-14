locals {
  # todo: change in upstream pr to scott
  # "git::git@github.com:username-goes-here/repository-name-goes-here.git"
  base_source_url    = "git::git@github.com:caerulescens/terraform-provider-minikube.git//examples/guides/terragrunt/terraform_project"
  ref                = "feature/terragrunt-usage"
  kubernetes_version = "v1.28.3"
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite"
  contents  = <<EOF
provider "minikube" {
  kubernetes_version = "${local.kubernetes_version}"
}
EOF
}

remote_state {
  backend = "local"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite"
  }
  config = {
    path = "${get_parent_terragrunt_dir()}/${path_relative_to_include()}/terraform.tfstate"
  }
}

inputs = {}
