#!/usr/bin/env bash

set -u

profile_prefix="terraform-provider-minikube-acc-"
minikube_home="${MINIKUBE_HOME:-$HOME/.minikube}"
if [[ "$(basename "$minikube_home")" != ".minikube" ]]; then
  minikube_home="$minikube_home/.minikube"
fi
profile_root="$minikube_home/profiles"

# Include names used before acceptance profiles received a dedicated prefix.
profile_dirs=(
  "$profile_root/$profile_prefix"*
  "$profile_root/TestClusterCreationDocker"
  "$profile_root/TestClusterCreationDockerUpdate"
  "$profile_root/TestClusterCreationDockerAddons"
  "$profile_root/TestClusterCreationQemu"
  "$profile_root/TestClusterCreationHyperV"
  "$profile_root/multinode"
  "$profile_root/ha"
)

for profile_dir in "${profile_dirs[@]}"; do
  [[ -d "$profile_dir" ]] || continue

  profile="$(basename "$profile_dir")"
  echo "Cleaning acceptance profile $profile"
  minikube delete --profile "$profile" || true
done
