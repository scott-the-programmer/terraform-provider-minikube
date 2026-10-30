# shellcheck shell=bash
#
# Thin wrappers around the terraform CLI. Each flavour gets its own copy of the
# stack under the work directory so state, plugin cache and lock file never
# collide between drivers running back to back.

export TF_IN_AUTOMATION=1
export TF_INPUT=0

tf::prepare() {
  local flavour=$1 work_dir=$2 stack_dir=$3 tfvars=$4 provider_version=$5
  local dir="$work_dir/$flavour"

  rm -rf "$dir"
  mkdir -p "$dir"
  cp "$stack_dir"/*.tf "$dir/"
  cp "$tfvars" "$dir/terraform.tfvars"

  if [[ "$provider_version" != "99.99.99" ]]; then
    sed -i.bak "s/version = \"99.99.99\"/version = \"$provider_version\"/" "$dir/versions.tf"
    rm -f "$dir/versions.tf.bak"
  fi

  printf '%s' "$dir"
}

tf::init() {
  local dir=$1
  terraform -chdir="$dir" init -no-color -upgrade
}

# Initialise from the public registry even when the caller has configured the
# local filesystem mirror used for the development build. Unsetting
# TF_CLI_CONFIG_FILE is not enough: 'make set-local' puts 99.99.99 under
# ~/.terraform.d/plugins, which terraform treats as an implicit mirror, and then
# it never asks the registry for this provider. An explicit direct-only config
# turns the implicit mirrors off.
tf::init_from_registry() {
  local dir=$1
  local cli_config="$dir/registry-only.tfrc"
  printf 'provider_installation {\n  direct {}\n}\n' >"$cli_config"
  TF_CLI_CONFIG_FILE="$cli_config" terraform -chdir="$dir" init -no-color -upgrade
}

tf::set_provider_version() {
  local dir=$1 version=$2
  sed -i.bak \
    "s/^\([[:space:]]*\)version = \"[^\"]*\"/\1version = \"$version\"/" \
    "$dir/versions.tf"
  rm -f "$dir/versions.tf.bak"
}

# Run a command with a deadline without relying on GNU timeout, which is not
# available on macOS. Returns 124 on timeout, matching GNU timeout.
tf::_with_timeout() {
  local timeout_s=$1
  shift

  local marker="${TMPDIR:-/tmp}/tf-minikube-e2e-timeout.$$.$RANDOM"
  local command_pid watchdog_pid rc
  rm -f "$marker"

  "$@" &
  command_pid=$!
  (
    # Poll rather than using one long sleep. On macOS, terminating a shell that
    # is waiting on sleep can leave the caller blocked until that sleep ends.
    local elapsed=0
    while ((elapsed < timeout_s)); do
      sleep 1
      kill -0 "$command_pid" 2>/dev/null || exit 0
      elapsed=$((elapsed + 1))
    done

    : >"$marker"
    kill -TERM "$command_pid" 2>/dev/null || true
    # Poll during the grace period so a command that handles TERM does not
    # make its caller wait for the full five seconds.
    local attempt
    for attempt in 1 2 3 4 5; do
      kill -0 "$command_pid" 2>/dev/null || exit 0
      sleep 1
    done
    kill -KILL "$command_pid" 2>/dev/null || true
  ) &
  watchdog_pid=$!

  if wait "$command_pid"; then
    rc=0
  else
    rc=$?
  fi
  kill "$watchdog_pid" 2>/dev/null || true
  wait "$watchdog_pid" 2>/dev/null || true

  if [[ -f "$marker" ]]; then
    rm -f "$marker"
    printf 'timed out after %ss waiting for: %s\n' "$timeout_s" "$*" >&2
    return 124
  fi
  rm -f "$marker"
  return "$rc"
}

tf::apply() {
  local dir=$1 timeout_s=$2
  tf::_with_timeout "$timeout_s" \
    terraform -chdir="$dir" apply -no-color -auto-approve
}

tf::destroy() {
  local dir=$1 timeout_s=${2:-900}
  tf::_with_timeout "$timeout_s" \
    terraform -chdir="$dir" destroy -no-color -auto-approve
}

# Succeeds only when a re-plan reports no changes. Exit code 2 from
# -detailed-exitcode means "there is a diff", which is a real failure here: the
# provider should have read back everything it wrote.
tf::plan_is_clean() {
  local dir=$1 out rc
  out=$(terraform -chdir="$dir" plan -no-color -detailed-exitcode 2>&1)
  rc=$?
  case $rc in
    0) return 0 ;;
    2)
      printf 'terraform plan reported a diff after apply:\n%s\n' "$out" >&2
      return 1
      ;;
    *)
      printf '%s\n' "$out" >&2
      return $rc
      ;;
  esac
}

tf::output() {
  local dir=$1 name=$2
  terraform -chdir="$dir" output -raw -no-color "$name"
}

# Number of resources left in state. Used to confirm a destroy really emptied it.
tf::state_count() {
  local dir=$1
  terraform -chdir="$dir" state list 2>/dev/null | grep -c . || true
}
