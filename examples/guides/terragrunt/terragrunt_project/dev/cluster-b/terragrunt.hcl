include "root" {
  path = find_in_parent_folders()
}

include "envcommon" {
  path   = "${dirname(find_in_parent_folders())}/_envcommon/minikube.hcl"
  expose = true
}

terraform {
  source = "${include.envcommon.locals.base_source_url}?ref=${include.envcommon.locals.ref}"
}

inputs = {
  cluster_name = "dev-cluster-b"
}
