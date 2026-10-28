#!/usr/bin/env bash
# Create with the registry release, then require a no-op with this checkout.
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/../.." && pwd)
export TF_IN_AUTOMATION=1 TF_INPUT=0

usage() {
  printf '%s\n' \
    'Usage: test/e2e/upgrade.sh' \
    'Requires terraform, curl, jq, timeout, a working Docker daemon, and Go/make.' \
    'UPGRADE_FROM_VERSION: registry version (default: latest registry release).' \
    'UPGRADE_PROVIDER_BINARY: existing branch binary (default: make build).' \
    'UPGRADE_TIMEOUT: timeout per Terraform operation in seconds (default: 1200).' \
    'Creates a unique test/e2e/.work/upgrade.* directory; always destroys the cluster.'
}
if [[ ${1:-} == --help || ${1:-} == -h ]]; then usage; exit 0; fi
if [[ $# != 0 ]]; then usage >&2; exit 2; fi
for tool in terraform curl jq timeout; do
  command -v "$tool" >/dev/null || { printf 'Missing tool: %s\n' "$tool" >&2; exit 1; }
done

version=${UPGRADE_FROM_VERSION:-$(curl --fail --silent --show-error \
  https://registry.terraform.io/v1/providers/scott-the-programmer/minikube | jq -er '.version')}
version=${version#v}
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ && $version != 99.99.99 ]] || {
  printf 'Expected a released provider version, got: %s\n' "$version" >&2
  exit 1
}

binary=${UPGRADE_PROVIDER_BINARY:-}
if [[ -z $binary ]]; then
  make -C "$REPO_ROOT" build
  binary="$REPO_ROOT/bin/terraform-provider-minikube"
fi
[[ -x $binary ]] || { printf 'Not executable: %s\n' "$binary" >&2; exit 1; }
platform=$(terraform version -json | jq -er '.platform')
mkdir -p "$SCRIPT_DIR/.work"
work=$(mktemp -d "$SCRIPT_DIR/.work/upgrade.XXXXXXXX")
stack="$work/stack"
mkdir -p "$stack" "$work/mirror/registry.terraform.io/scott-the-programmer/minikube/99.99.99/$platform"
cp "$binary" "$work/mirror/registry.terraform.io/scott-the-programmer/minikube/99.99.99/$platform/terraform-provider-minikube"
# Explicit configs bypass user dev_overrides, implied mirrors and plugin caches.
printf 'provider_installation {\n  direct {}\n}\n' > "$work/released.tfrc"
printf 'provider_installation {\n  filesystem_mirror {\n    path = "%s"\n  }\n}\n' "$work/mirror" > "$work/branch.tfrc"
unset TF_PLUGIN_CACHE_DIR
# External CLI flags must not disable refresh or target only part of the stack.
for name in ${!TF_CLI_ARGS*}; do unset "$name"; done
export TF_CLI_CONFIG_FILE="$work/released.tfrc"

select_version() {
  printf 'terraform {\n  required_providers {\n    minikube = {\n      source = "scott-the-programmer/minikube"\n      version = "= %s"\n    }\n  }\n}\n' "$1" > "$stack/versions.tf"
}
tf() {
  timeout --foreground "${UPGRADE_TIMEOUT:-1200}" terraform -chdir="$stack" "$@"
}
cleanup_needed=0
installed_version=$version
installed_config=$TF_CLI_CONFIG_FILE
cleanup() {
  local rc=$?
  trap - EXIT
  if [[ $cleanup_needed == 1 ]]; then
    # If branch init failed, use the last successfully installed provider.
    select_version "$installed_version"
    export TF_CLI_CONFIG_FILE="$installed_config"
    if ! tf destroy -no-color -auto-approve > "$work/destroy.log" 2>&1; then
      printf 'ERROR: cleanup failed; see %s/destroy.log\n' "$work" >&2
      rc=1
    fi
  fi
  printf 'Upgrade test artifacts (may contain credentials): %s\n' "$work"
  exit "$rc"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

cp "$SCRIPT_DIR/stack/"*.tf "$stack/"
cp "$SCRIPT_DIR/flavours/docker.tfvars" "$stack/terraform.tfvars"
# Isolate the minikube profile from normal E2E runs and concurrent upgrades.
printf '\ncluster_name = "tf-minikube-%s"\n' "${work##*/}" > "$stack/upgrade.auto.tfvars"
# Pin Kubernetes as well: provider releases can embed different minikube defaults.
printf 'kubernetes_version = "v1.34.0"\n' >> "$stack/upgrade.auto.tfvars"
select_version "$version"
printf 'Creating stack with released provider %s\n' "$version"
tf init -no-color | tee "$work/released-init.log"
cleanup_needed=1
tf apply -no-color -auto-approve | tee "$work/released-apply.log"
# Establish that any subsequent diff is caused by the provider upgrade.
tf plan -no-color -detailed-exitcode | tee "$work/released-plan.log"
cp "$stack/terraform.tfstate" "$work/released.tfstate"
cp "$stack/.terraform.lock.hcl" "$work/released.lock.hcl"

printf 'Switching the same stack and state to the branch provider\n'
select_version 99.99.99
export TF_CLI_CONFIG_FILE="$work/branch.tfrc"
tf init -no-color -upgrade | tee "$work/branch-init.log"
installed_version=99.99.99
installed_config=$TF_CLI_CONFIG_FILE
# With pipefail, both an error (1) and a diff (2) fail before any apply.
tf plan -no-color -detailed-exitcode -out=upgrade.tfplan | tee "$work/branch-plan.log"
# Apply precisely the verified no-op plan, never an unverified fresh plan.
tf apply -no-color -auto-approve upgrade.tfplan | tee "$work/branch-apply.log"
tf plan -no-color -detailed-exitcode | tee "$work/branch-post-apply-plan.log"
printf 'PASS: released provider %s -> branch is a no-op\n' "$version"
