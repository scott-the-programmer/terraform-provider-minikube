# shellcheck shell=bash
#
# Logging and result tracking shared by the e2e scripts.

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  _C_RED=$'\033[31m'
  _C_GREEN=$'\033[32m'
  _C_YELLOW=$'\033[33m'
  _C_BLUE=$'\033[34m'
  _C_BOLD=$'\033[1m'
  _C_OFF=$'\033[0m'
else
  _C_RED='' _C_GREEN='' _C_YELLOW='' _C_BLUE='' _C_BOLD='' _C_OFF=''
fi

_ts() { date -u '+%H:%M:%S'; }

log::info() { printf '%s %s==>%s %s\n' "$(_ts)" "$_C_BLUE" "$_C_OFF" "$*"; }
log::warn() { printf '%s %sWARN%s %s\n' "$(_ts)" "$_C_YELLOW" "$_C_OFF" "$*" >&2; }
log::error() { printf '%s %sERROR%s %s\n' "$(_ts)" "$_C_RED" "$_C_OFF" "$*" >&2; }
log::pass() { printf '%s %sPASS%s %s\n' "$(_ts)" "$_C_GREEN" "$_C_OFF" "$*"; }
log::fail() { printf '%s %sFAIL%s %s\n' "$(_ts)" "$_C_RED" "$_C_OFF" "$*" >&2; }
log::skip() { printf '%s %sSKIP%s %s\n' "$(_ts)" "$_C_YELLOW" "$_C_OFF" "$*"; }

log::banner() {
  printf '\n%s%s%s\n' "$_C_BOLD" "$(printf '=%.0s' {1..72})" "$_C_OFF"
  printf '%s%s%s\n' "$_C_BOLD" "$*" "$_C_OFF"
  printf '%s%s%s\n\n' "$_C_BOLD" "$(printf '=%.0s' {1..72})" "$_C_OFF"
}

# Indents whatever it is piped, a line at a time. Deliberately not `sed`: sed
# block-buffers when its stdout is a file rather than a terminal, which hides
# the progress of a long terraform apply when the run is being logged.
log::indent() {
  local line
  while IFS= read -r line || [[ -n $line ]]; do
    printf '    %s\n' "$line"
  done
}

# Run a command, streaming its output indented so it is distinguishable from
# the harness's own logging.
log::run() {
  log::info "\$ $*"
  "$@" 2>&1 | log::indent
  return "${PIPESTATUS[0]}"
}
