# shellcheck shell=bash
#
# Assertion and retry helpers. Checks record their outcome instead of aborting,
# so one broken assertion still lets the rest of the suite report.

# Every result recorded across the whole run: "<flavour>|<status>|<name>".
E2E_RESULTS=()
E2E_FAILED=0

assert::record() {
  local status=$1 name=$2
  E2E_RESULTS+=("${E2E_CURRENT_FLAVOUR:-?}|${status}|${name}")
  case "$status" in
    PASS) log::pass "$name" ;;
    FAIL)
      log::fail "$name"
      E2E_FAILED=$((E2E_FAILED + 1))
      ;;
    SKIP) log::skip "$name" ;;
  esac
}

# check "<name>" <command...>
#
# Runs the command with output captured. On failure the output is replayed so
# the log says why. Returns the command's exit status.
check() {
  local name=$1
  shift
  local out rc
  out=$("$@" 2>&1)
  rc=$?
  if [[ $rc -eq 0 ]]; then
    assert::record PASS "$name"
  else
    assert::record FAIL "$name"
    [[ -n $out ]] && printf '%s\n' "$out" | log::indent >&2
  fi
  return $rc
}

# check::stream "<name>" <command...>
#
# Same as check, but streams output as it happens. Used for the terraform steps,
# which can run for many minutes and are unpleasant to watch in silence.
check::stream() {
  local name=$1
  shift
  local rc
  "$@" 2>&1 | log::indent
  rc=${PIPESTATUS[0]}
  if [[ $rc -eq 0 ]]; then
    assert::record PASS "$name"
  else
    assert::record FAIL "$name"
  fi
  return $rc
}

# check::skip "<name>" "<reason>"
check::skip() {
  assert::record SKIP "$1${2:+ - $2}"
}

assert::equals() {
  local expected=$1 actual=$2
  if [[ "$expected" != "$actual" ]]; then
    printf 'expected %q, got %q\n' "$expected" "$actual" >&2
    return 1
  fi
}

assert::contains() {
  local haystack=$1 needle=$2
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'expected output to contain %q, got:\n%s\n' "$needle" "$haystack" >&2
    return 1
  fi
}

assert::not_empty() {
  if [[ -z "${1//[[:space:]]/}" ]]; then
    printf 'expected a non-empty value\n' >&2
    return 1
  fi
}

# retry::until <timeout_seconds> <interval_seconds> <command...>
#
# Retries until the command succeeds or the timeout expires. The last attempt's
# output is emitted on failure.
retry::until() {
  local timeout=$1 interval=$2
  shift 2
  local deadline=$((SECONDS + timeout)) out rc
  while :; do
    out=$("$@" 2>&1)
    rc=$?
    [[ $rc -eq 0 ]] && {
      printf '%s' "$out"
      return 0
    }
    if ((SECONDS >= deadline)); then
      printf 'timed out after %ss waiting for: %s\n%s\n' "$timeout" "$*" "$out" >&2
      return 1
    fi
    sleep "$interval"
  done
}

assert::summary() {
  local total=${#E2E_RESULTS[@]} passed=0 failed=0 skipped=0 entry
  log::banner "SUMMARY"
  for entry in "${E2E_RESULTS[@]}"; do
    local flavour=${entry%%|*}
    local rest=${entry#*|}
    local status=${rest%%|*}
    local name=${rest#*|}
    printf '  %-8s %-6s %s\n' "$flavour" "$status" "$name"
    case "$status" in
      PASS) passed=$((passed + 1)) ;;
      FAIL) failed=$((failed + 1)) ;;
      SKIP) skipped=$((skipped + 1)) ;;
    esac
  done
  printf '\n  %d checks: %d passed, %d failed, %d skipped\n\n' \
    "$total" "$passed" "$failed" "$skipped"
  [[ $failed -eq 0 ]]
}
