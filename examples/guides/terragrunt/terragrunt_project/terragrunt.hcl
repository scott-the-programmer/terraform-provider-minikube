locals {
  # load variables
  environment_vars  = read_terragrunt_config(find_in_parent_folders("environment.hcl"))
}

# merge variables
inputs = merge(
  local.environment_vars.locals,
)
