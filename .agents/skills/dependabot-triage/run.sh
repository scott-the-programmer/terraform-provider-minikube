#!/usr/bin/env bash
#
# Run the dependabot-triage skill inside an isolated container.
# Mounts host creds read-only where possible. Shares the docker daemon so
# `make schema-container` still works for k8s.io/minikube bumps.
#
# Usage:
#   ./run.sh                       # headless: runs /dependabot-triage and exits
#   ./run.sh --interactive         # drops into an interactive claude session
#   ./run.sh --rebuild              # force image rebuild
#
# Required on host:
#   - docker
#   - gh CLI authenticated (or GITHUB_TOKEN env var exported)
#   - claude CLI authenticated at $HOME/.claude (for session reuse)

set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SKILL_DIR/../../.." && pwd)"
IMAGE_TAG="tpm-dependabot-triage:latest"

REBUILD=0
INTERACTIVE=0
for arg in "$@"; do
  case "$arg" in
    --rebuild)      REBUILD=1 ;;
    --interactive)  INTERACTIVE=1 ;;
    -h|--help)
      sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "unknown arg: $arg" >&2; exit 2 ;;
  esac
done

# --- build image if missing or --rebuild ---
if [[ $REBUILD -eq 1 ]] || ! docker image inspect "$IMAGE_TAG" >/dev/null 2>&1; then
  echo "building $IMAGE_TAG..."
  docker build -t "$IMAGE_TAG" "$SKILL_DIR"
fi

# --- resolve github token ---
: "${GITHUB_TOKEN:=${GH_TOKEN:-}}"
if [[ -z "$GITHUB_TOKEN" ]]; then
  if command -v gh >/dev/null 2>&1; then
    GITHUB_TOKEN="$(gh auth token 2>/dev/null || true)"
  fi
fi
if [[ -z "$GITHUB_TOKEN" ]]; then
  echo "error: no GITHUB_TOKEN. export one, or run 'gh auth login' on the host." >&2
  exit 1
fi

# --- cred paths ---
CLAUDE_DIR="${CLAUDE_DIR:-$HOME/.claude}"
if [[ ! -d "$CLAUDE_DIR" ]]; then
  echo "error: $CLAUDE_DIR not found. Is claude code installed/authed on host?" >&2
  exit 1
fi

GITCONFIG_MOUNT=()
if [[ -f "$HOME/.gitconfig" ]]; then
  GITCONFIG_MOUNT=(-v "$HOME/.gitconfig:/root/.gitconfig:ro")
fi

# --- docker socket ---
# macOS/Linux Docker Desktop uses /var/run/docker.sock. Mount it so the
# container can spawn sibling containers (needed by make schema-container).
DOCKER_SOCK="/var/run/docker.sock"
if [[ ! -S "$DOCKER_SOCK" ]]; then
  echo "warn: $DOCKER_SOCK not found. make schema-container will fail for minikube bumps." >&2
fi

# --- workspace mount ---
# IMPORTANT: mount the repo at its *host* path inside the container. When the
# skill runs `make schema-container`, that spawns a sibling container via the
# host docker daemon, and docker resolves -v paths against the host filesystem.
# If the in-container path differs, bind mounts in the child container break.
WORKSPACE_MOUNT=(-v "$REPO_ROOT:$REPO_ROOT")
WORKDIR="$REPO_ROOT"

# --- run ---
DOCKER_ARGS=(
  run --rm
  "${WORKSPACE_MOUNT[@]}"
  -v "$CLAUDE_DIR:/root/.claude"
  -v "$DOCKER_SOCK:$DOCKER_SOCK"
  "${GITCONFIG_MOUNT[@]}"
  -e GITHUB_TOKEN
  -e GH_TOKEN="$GITHUB_TOKEN"
  -w "$WORKDIR"
)

if [[ $INTERACTIVE -eq 1 || -t 0 ]]; then
  DOCKER_ARGS+=(-it)
fi

if [[ $INTERACTIVE -eq 1 ]]; then
  exec docker "${DOCKER_ARGS[@]}" "$IMAGE_TAG"
else
  # Headless: runs /dependabot-triage, prints summary, exits.
  exec docker "${DOCKER_ARGS[@]}" "$IMAGE_TAG" \
    -p "/dependabot-triage" \
    --permission-mode acceptEdits
fi
