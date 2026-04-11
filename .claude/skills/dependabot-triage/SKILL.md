---
name: dependabot-triage
description: >
  Triage and auto-merge Dependabot PRs for terraform-provider-minikube. Lists open
  dependabot PRs, checks CI status via gh, classifies semver bump (patch/minor/major),
  auto-merges safe updates, and flags the rest with a reason. Special-cases
  k8s.io/minikube bumps (requires schema regeneration). Use when user says
  "triage dependabot", "merge dependabot PRs", "chunk through dependabot",
  "dependabot queue", or invokes /dependabot-triage.
---

Automate the dependabot PR grind. Never blind-merge — always gate on CI + semver classification + special-case rules.

## Required tools

- `gh` CLI authenticated with repo write access
- `git` (for schema checks on minikube bumps)
- Repo's `make schema-container` target (see `MEMORY.md` — containerized schema gen)

## Containerized runtime (preferred)

This skill ships with its own sandbox so you don't pollute the host:

```bash
./.claude/skills/dependabot-triage/run.sh               # headless
./.claude/skills/dependabot-triage/run.sh --interactive # drop into a shell
./.claude/skills/dependabot-triage/run.sh --rebuild     # rebuild image
```

The runner mounts:
- repo at the **same path** inside + outside the container (so sibling
  containers spawned by `make schema-container` resolve bind mounts correctly)
- `$HOME/.claude` → `/root/.claude` (claude session + global skills)
- `/var/run/docker.sock` (sibling container spawning)
- `$HOME/.gitconfig` (read-only)
- `GITHUB_TOKEN` env (auto-extracted via `gh auth token` if unset)

If running on the host directly (not in the container), just ensure `gh`,
`git`, `make`, and `docker` are on PATH.

## Workflow

Run these in order. Stop and report if any step fails in a way that needs human judgment.

### 1. Enumerate open dependabot PRs

```bash
gh pr list --author "app/dependabot" \
  --json number,title,headRefName,createdAt,mergeable,mergeStateStatus,statusCheckRollup \
  --limit 50
```

Group results into buckets before acting. Do not process one-by-one without the full picture — batching lets you spot related bumps (e.g. multiple aws-sdk-go-v2 submodules) that should be merged together or rebased after the first lands.

### 2. Classify each PR

Parse the title `Bump <pkg> from X.Y.Z to A.B.C`:

| Bump type | Rule | Condition |
|---|---|---|
| **patch** (`X.Y.Z → X.Y.Z'`) | auto-merge | CI green |
| **minor** (`X.Y.Z → X.Y'.0`) | auto-merge | CI green AND not in `high-risk` list |
| **major** (`X.Y.Z → X'.0.0`) | **flag, never auto-merge** | always requires human review |
| **pre-1.0** (`0.x.y`) | treat minor as major | SemVer guarantees don't apply |
| **+incompatible** Go module | treat as major | signals intentional break |

**high-risk packages** (minor bumps need human eyes):
- `k8s.io/minikube` — schema drift, always run schema verification
- `github.com/hashicorp/terraform-plugin-sdk/v2` — provider contract surface
- `github.com/hashicorp/terraform-plugin-framework` — same
- `k8s.io/*` (client-go, api, apimachinery) — API churn

### 3. Gate on CI

For each candidate to merge:

```bash
gh pr checks <number> --json name,state,conclusion
```

Only proceed if **all** required checks are `SUCCESS`. Rules:
- `PENDING` / `IN_PROGRESS` → skip this pass, report "waiting on CI"
- `FAILURE` → **do not retry blindly**. Read the failure log:
  ```bash
  gh run view --log-failed --job=<job-id>
  ```
  Common causes + responses documented in §6.
- `SKIPPED` → treat as pass if check is not required.

### 4. Special case: k8s.io/minikube bumps

These regenerate the provider schema. Workflow:

1. Check out the PR branch locally: `gh pr checkout <number>`
2. Extract target version from title (e.g. `v1.38.1`)
3. Run containerized schema gen: `make schema-container MINIKUBE_VERSION=v<version>`
4. `git diff minikube/schema_cluster.go`
5. **If no diff** → schema already current, safe to merge after CI.
6. **If diff** → DO NOT auto-merge. Report the diff to user, ask whether to commit the regenerated schema to the PR branch.
7. Alternatively, trigger the existing workflow instead of running locally:
   ```bash
   gh workflow run schema-verification.yml -f minikube_version=v<version>
   ```

### 5. Auto-merge safe PRs

For each PR that passes all gates:

```bash
gh pr merge <number> --squash --auto --delete-branch
```

Use `--auto` so if CI is still ticking, GitHub merges when it turns green. Use `--squash` to keep history linear (matches repo convention — check `git log --oneline -20` if unsure).

**Never** use `--admin` to bypass required checks. **Never** use `--rebase` on a dependabot branch unless user asks (dependabot owns the branch and will re-push).

### 6. CI failure triage

When CI is red, classify before acting:

| Symptom | Likely cause | Action |
|---|---|---|
| `go.sum` mismatch | stale sum file | Comment `@dependabot rebase` on PR |
| Merge conflict with main | another dep landed first | Comment `@dependabot recreate` |
| Test failure referencing the bumped pkg | real regression | **flag for human** — do not rebase |
| Flaky codecov upload | infra, not code | Comment `@dependabot rebase` once; if still red, flag |
| Schema drift on minikube bump | upstream schema change | Follow §4 |

Dependabot commands — comment on the PR, don't close/reopen:
- `@dependabot rebase` — rebase on main
- `@dependabot recreate` — regenerate PR from scratch
- `@dependabot merge` — merge after CI (alternative to `gh pr merge`)
- `@dependabot close` — abandon

### 7. Report

Produce a single summary block at the end. Format:

```
Dependabot triage: N PRs

Merged (auto):
  #239 go-getter 1.8.4→1.8.6  [patch, CI green]
  #237 aws-sdk-go-v2/s3 1.95.0→1.97.3  [minor, CI green]

Queued (--auto, waiting on CI):
  #238 otel/sdk 1.39.0→1.43.0  [minor]

Flagged (needs review):
  #233 docker/cli 28.4.0→29.2.0+incompatible  [major, +incompatible]
  #228 k8s.io/minikube 1.37.0→1.38.1  [high-risk, schema diff — see §4]

Rebased (conflicts/stale sum):
  #232 terraform-plugin-sdk/v2 2.38.2→2.39.0  [commented rebase]

Failed (human needed):
  #234 grpc 1.78.0→1.79.3  [test failure in foo_test.go:42]
```

Keep it copy-pasteable. No prose summary beyond this block.

## Rules of engagement

- **Never force-merge.** `--admin` and `--no-verify` are off-limits without explicit user approval each time.
- **Never close a PR** unless user asks — dependabot will recreate endlessly if the dep is still outdated.
- **Never bump multiple majors in parallel** without user sign-off. Land them one at a time so bisect still works.
- **Don't touch `go.mod` / `go.sum` directly.** Let dependabot rebase.
- **Don't run `make test` locally** unless CI is unreachable or user asks — CI is authoritative and cheaper.
- **Respect the merge window.** If user has announced a freeze (check recent conversation or memory), flag everything and merge nothing.

## Boundaries

This skill triages and merges. It does **not**:
- Upgrade deps proactively (that's dependabot's job)
- Write release notes
- Bump the provider version (release workflow handles that)
- Modify `.github/dependabot.yml` groupings without user request

User says "stop" mid-run: finish the PR in flight, report state, halt.
