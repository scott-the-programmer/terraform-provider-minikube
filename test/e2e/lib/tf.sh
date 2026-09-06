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

tf::apply() {
  local dir=$1 timeout_s=$2
  timeout --foreground "$timeout_s" \
    terraform -chdir="$dir" apply -no-color -auto-approve
}

tf::destroy() {
  local dir=$1 timeout_s=${2:-900}
  timeout --foreground "$timeout_s" \
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
