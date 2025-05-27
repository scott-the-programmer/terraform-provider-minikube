locals {
  base_source_url    = "git::https://github.com/scott-the-programmer/terraform-provider-minikube.git//examples/guides/terragrunt/terraform_project"
  ref                = "main"
  kubernetes_version = "v1.30.2"
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
