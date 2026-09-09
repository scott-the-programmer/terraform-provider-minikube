#!/usr/bin/env bash
#
# End-to-end tests for terraform-provider-minikube.
#
# Creates a minikube cluster per driver flavour with terraform, connects to it
# with kubectl using nothing but the provider's own outputs, deploys nginx and
# proves the whole path works, then tears everything back down.
#
# Deliberately implemented in bash + terraform + kubectl only: nothing here
# needs the Go toolchain or the provider's test harness.
#
# Usage: ./run.sh [options] [flavour ...]
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/../.." && pwd)

# shellcheck source=lib/log.sh
source "$SCRIPT_DIR/lib/log.sh"
# shellcheck source=lib/assert.sh
source "$SCRIPT_DIR/lib/assert.sh"
# shellcheck source=lib/tf.sh
source "$SCRIPT_DIR/lib/tf.sh"
# shellcheck source=lib/kube.sh
source "$SCRIPT_DIR/lib/kube.sh"
# shellcheck source=lib/checks.sh
source "$SCRIPT_DIR/lib/checks.sh"

STACK_DIR="$SCRIPT_DIR/stack"
FLAVOUR_DIR="$SCRIPT_DIR/flavours"
MANIFEST="$SCRIPT_DIR/manifests/nginx.yaml"

WORK_DIR=${E2E_WORK_DIR:-"$SCRIPT_DIR/.work"}
PROVIDER_VERSION=${E2E_PROVIDER_VERSION:-99.99.99}
BUILD_PROVIDER=1
KEEP_CLUSTERS=0
PRELOAD_IMAGE=${E2E_PRELOAD_IMAGE:-0}
REQUESTED_FLAVOURS=()

E2E_PORT_FORWARD_PID=''
E2E_PORT_FORWARD_PORT=''
E2E_KUBECONFIG=''
E2E_CURRENT_FLAVOUR=''
CLEANUP_DIRS=()

usage() {
  cat <<EOF
Usage: $(basename "$0") [options] [flavour ...]

Flavours are the file names under flavours/ (docker, qemu, podman). With none
given, every flavour is attempted; ones whose driver is not installed on this
machine are reported as SKIP rather than failing the run.

Options:
  -k, --keep          Leave the clusters and terraform state in place on exit.
  -s, --skip-build    Do not rebuild/install the local provider first.
  -p, --preload-image Side-load the workload image from the host into the
                      cluster instead of letting the cluster pull it. Useful
                      offline, behind a registry rate limit, or where the
                      cluster has no DNS out. Needs the minikube CLI.
  -w, --work-dir DIR  Where to materialise per-flavour terraform dirs.
                      (default: $WORK_DIR)
  -l, --list          List the available flavours and exit.
  -h, --help          Show this help.

Environment:
  E2E_PROVIDER_VERSION  Provider version to test. Defaults to 99.99.99, the
                        version 'make set-local' publishes to the local
                        filesystem mirror. Set it to a released version to run
                        the suite against the registry build instead.
  TF_CLI_CONFIG_FILE    Honoured if already set; otherwise the local mirror
                        config is used when testing 99.99.99.
  E2E_PRELOAD_IMAGE     Set to 1 for the same effect as --preload-image.
  E2E_WORK_DIR          Same as --work-dir.

Examples:
  ./run.sh                     # everything that can run on this machine
  ./run.sh docker              # just the docker driver
  ./run.sh --keep docker qemu  # leave both clusters up for inspection
EOF
}

list_flavours() {
  local f
  for f in "$FLAVOUR_DIR"/*.tfvars; do
    f=$(basename "$f" .tfvars)
    # shellcheck disable=SC1090
    (source "$FLAVOUR_DIR/$f.env" && printf '  %-8s %s\n' "$f" "$E2E_DESCRIPTION")
  done
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -k | --keep) KEEP_CLUSTERS=1 ;;
      -s | --skip-build) BUILD_PROVIDER=0 ;;
      -p | --preload-image) PRELOAD_IMAGE=1 ;;
      -w | --work-dir)
        WORK_DIR=$2
        shift
        ;;
      -l | --list)
        list_flavours
        exit 0
        ;;
      -h | --help)
        usage
        exit 0
        ;;
      -*)
        log::error "unknown option: $1"
        usage >&2
        exit 2
        ;;
      *) REQUESTED_FLAVOURS+=("$1") ;;
    esac
    shift
  done
}

require_host_tools() {
  local missing=() tool
  for tool in terraform kubectl curl base64 timeout; do
    command -v "$tool" >/dev/null 2>&1 || missing+=("$tool")
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    log::error "missing required tools: ${missing[*]}"
    exit 1
  fi
}

# Returns 0 when every requirement of the flavour is satisfied. Requirements are
# space separated; "a|b" inside one means either will do.
flavour_is_runnable() {
  local group found alt alts
  for group in $E2E_REQUIRES; do
    found=0
    IFS='|' read -r -a alts <<<"$group"
    for alt in "${alts[@]}"; do
      command -v "$alt" >/dev/null 2>&1 && {
        found=1
        break
      }
    done
    if [[ $found -eq 0 ]]; then
      printf '%s not installed' "${group//|/ or }"
      return 1
    fi
  done

  if [[ -n ${E2E_HEALTHCHECK:-} ]]; then
    if ! eval "$E2E_HEALTHCHECK" >/dev/null 2>&1; then
      printf '"%s" failed' "$E2E_HEALTHCHECK"
      return 1
    fi
  fi
  return 0
}

build_local_provider() {
  log::info "building and installing the local provider (version $PROVIDER_VERSION)"
  log::run make -C "$REPO_ROOT" set-local
}

configure_provider_source() {
  if [[ "$PROVIDER_VERSION" == "99.99.99" && -z "${TF_CLI_CONFIG_FILE:-}" ]]; then
    local cli_config="$REPO_ROOT/bin/terraform-local.tfrc"
    if [[ ! -f "$cli_config" ]]; then
      log::info "generating the local provider CLI config"
      log::run make -C "$REPO_ROOT" local-cli-config
    fi
    export TF_CLI_CONFIG_FILE="$cli_config"
  fi
  [[ -n ${TF_CLI_CONFIG_FILE:-} ]] &&
    log::info "using terraform CLI config $TF_CLI_CONFIG_FILE"
  return 0
}

on_signal() {
  log::warn "interrupted; running cleanup"
  exit 130
}

# The image the manifests ask for, so there is one source of truth for it.
workload_image() {
  sed -n 's/^ *image: *//p' "$MANIFEST" | head -1
}

# Side-load the workload image from the host's container store into the cluster,
# for runs where the cluster itself cannot reach a registry. Driver-agnostic:
# minikube handles getting the tarball into whichever runtime is in use.
preload_image() {
  local profile=$1 image=$2 tar=$3

  command -v minikube >/dev/null 2>&1 || {
    printf 'the minikube CLI is needed to preload images\n' >&2
    return 1
  }

  local puller=''
  if command -v docker >/dev/null 2>&1; then
    puller=docker
  elif command -v podman >/dev/null 2>&1; then
    puller=podman
  else
    printf 'need docker or podman on the host to fetch %s\n' "$image" >&2
    return 1
  fi

  "$puller" image inspect "$image" >/dev/null 2>&1 || "$puller" pull "$image" || return 1
  "$puller" save "$image" -o "$tar" || return 1
  minikube -p "$profile" image load "$tar"
}

cleanup() {
  local rc=$?
  kube::stop_port_forward

  if [[ $KEEP_CLUSTERS -eq 1 ]]; then
    [[ ${#CLEANUP_DIRS[@]} -gt 0 ]] &&
      log::warn "--keep set; clusters left running. Tear down with:" &&
      printf '    terraform -chdir=%s destroy -auto-approve\n' "${CLEANUP_DIRS[@]}" >&2
    return $rc
  fi

  # Only reached for dirs a flavour did not tear down itself, i.e. after an
  # interrupt or an early failure. Leaving a cluster behind wedges the next run.
  local dir
  for dir in "${CLEANUP_DIRS[@]:-}"; do
    [[ -d "$dir" ]] || continue
    log::warn "cleaning up leftover cluster in $dir"
    tf::destroy "$dir" 900 >/dev/null 2>&1 || log::warn "cleanup of $dir failed"
  done
  return $rc
}

forget_cleanup() {
  local target=$1 dir
  local remaining=()
  for dir in "${CLEANUP_DIRS[@]:-}"; do
    [[ -z "$dir" || "$dir" == "$target" ]] && continue
    remaining+=("$dir")
  done
  if [[ ${#remaining[@]} -gt 0 ]]; then
    CLEANUP_DIRS=("${remaining[@]}")
  else
    CLEANUP_DIRS=()
  fi
}

run_flavour() {
  local flavour=$1
  E2E_CURRENT_FLAVOUR="$flavour"

  local tfvars="$FLAVOUR_DIR/$flavour.tfvars"
  local envfile="$FLAVOUR_DIR/$flavour.env"
  if [[ ! -f "$tfvars" || ! -f "$envfile" ]]; then
    log::error "unknown flavour '$flavour'; available:"
    list_flavours >&2
    return 1
  fi

  # shellcheck disable=SC1090
  source "$envfile"

  log::banner "FLAVOUR: $flavour - $E2E_DESCRIPTION"

  local reason
  if ! reason=$(flavour_is_runnable); then
    check::skip "$flavour driver available" "$reason"
    return 0
  fi

  local tf_dir
  tf_dir=$(tf::prepare "$flavour" "$WORK_DIR" "$STACK_DIR" "$tfvars" "$PROVIDER_VERSION")
  CLEANUP_DIRS+=("$tf_dir")

  E2E_KUBECONFIG="$tf_dir/kubeconfig"

  local cluster_name expected_nodes
  cluster_name=$(sed -n 's/^cluster_name *= *"\(.*\)"/\1/p' "$tfvars")
  expected_nodes=$(sed -n 's/^nodes *= *\([0-9]\+\).*/\1/p' "$tfvars")
  expected_nodes=${expected_nodes:-1}

  if ! check::stream "terraform init" tf::init "$tf_dir"; then
    return 1
  fi

  if ! check::stream "terraform apply creates the $flavour cluster" \
    tf::apply "$tf_dir" "${E2E_APPLY_TIMEOUT:-1200}"; then
    teardown_flavour "$tf_dir" || true
    return 1
  fi

  check "provider outputs expose a usable apiserver endpoint and credentials" \
    checks::outputs_usable "$tf_dir" || true

  check "terraform plan is clean after apply" tf::plan_is_clean "$tf_dir" || true

  if ! check "kubeconfig can be built from the provider outputs" \
    kube::write_kubeconfig "$tf_dir" "$E2E_KUBECONFIG" "$cluster_name"; then
    teardown_flavour "$tf_dir" || true
    return 1
  fi

  check "kubectl reaches the apiserver with the provider's credentials" \
    checks::apiserver_reachable || true
  check "all $expected_nodes node(s) report Ready" \
    checks::nodes_ready "$expected_nodes" || true
  check "kube-system pods are ready" checks::system_pods_ready || true
  check "the storage-provisioner addon installed a default storage class" \
    checks::default_storageclass || true

  if [[ $PRELOAD_IMAGE -eq 1 ]]; then
    local image
    image=$(workload_image)
    check::stream "preload $image into the cluster" \
      preload_image "$cluster_name" "$image" "$tf_dir/image.tar" || true
  fi

  check "nginx manifests apply" checks::apply_manifests "$MANIFEST" || true
  check "nginx deployment rolls out" checks::rollout_complete || true
  check "2 nginx pods are running" checks::pods_running 2 || true
  check "the service has 2 ready endpoints" checks::endpoints_ready 2 || true
  check "cluster DNS resolves the nginx service" checks::cluster_dns || true
  check "nginx answers in-cluster over the service" checks::in_cluster_http || true
  check "nginx answers on the host through a port-forward" \
    checks::port_forward_http "$tf_dir/port-forward.log" || true

  local node_ip
  node_ip=$(kube::node_ip 2>/dev/null || true)
  if checks::nodeport_reachable "$node_ip"; then
    check "nginx answers on the node port at $node_ip:30080" \
      checks::nodeport_http "$node_ip" || true
  else
    check::skip "nginx answers on the node port" \
      "node IP ${node_ip:-unknown} is not routable from the host with the $flavour driver"
  fi

  check "scaling the deployment to 3 updates the service endpoints" \
    checks::scale_to 3 || true
  check "the workload can be deleted again" checks::workload_removed || true

  teardown_flavour "$tf_dir"
}

teardown_flavour() {
  local tf_dir=$1

  if [[ $KEEP_CLUSTERS -eq 1 ]]; then
    check::skip "terraform destroy removes the cluster" "--keep was set"
    return 0
  fi

  if check::stream "terraform destroy removes the cluster" tf::destroy "$tf_dir" 900; then
    forget_cleanup "$tf_dir"
    check "the cluster is gone after destroy" checks::cluster_gone "$tf_dir" || true
  else
    return 1
  fi
}

main() {
  parse_args "$@"
  require_host_tools

  if [[ ${#REQUESTED_FLAVOURS[@]} -eq 0 ]]; then
    local f
    for f in "$FLAVOUR_DIR"/*.tfvars; do
      REQUESTED_FLAVOURS+=("$(basename "$f" .tfvars)")
    done
  fi

  mkdir -p "$WORK_DIR"
  # The signal handler only needs to exit; the EXIT trap does the actual work,
  # so an interrupt still tears down whatever clusters are up.
  trap cleanup EXIT
  trap on_signal INT TERM

  if [[ $BUILD_PROVIDER -eq 1 && "$PROVIDER_VERSION" == "99.99.99" ]]; then
    build_local_provider
  fi
  configure_provider_source

  log::info "flavours to run: ${REQUESTED_FLAVOURS[*]}"

  local flavour
  for flavour in "${REQUESTED_FLAVOURS[@]}"; do
    run_flavour "$flavour" || log::error "flavour '$flavour' did not complete"
  done

  E2E_CURRENT_FLAVOUR=''
  assert::summary
}

main "$@"
