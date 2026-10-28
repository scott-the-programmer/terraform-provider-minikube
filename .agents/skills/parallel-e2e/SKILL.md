---
name: parallel-e2e
description: >
  Run terraform-provider-minikube end-to-end tests and stress isolated worker
  processes at maximum host-safe parallelism. Use when asked to run parallel
  E2E tests, test parallel cluster creation, stress worker isolation, or run
  E2E with max parallelism. Exercises the actual locally built provider,
  verifies overlapping cluster lifecycles and settings isolation, and cleans up.
---

# Maximum-parallelism E2E

Run the tests, do not merely propose a plan. Work from the repository root
(`../../..` relative to this skill directory). Do not commit, push, or change
tracked code unless asked. Creating this skill does not itself authorize a test
run; execute this workflow when the user requests testing.

## 1. Inspect and prepare

Read these files before running commands; adapt to their current interfaces:

- `Makefile`
- `test/e2e/README.md` and `test/e2e/run.sh`
- `test/e2e/lib/{tf,kube,checks,assert}.sh`
- `test/e2e/stack/*.tf` and the selected `test/e2e/flavours/*` files
- `minikube/worker/{parent,protocol}.go`

Record the commit, dirty working-tree status, OS, CPU count, available memory,
free disk, Docker/driver health, Terraform version, and pinned Minikube module
version (`go list -m k8s.io/minikube`). Check required tools: Go, make, Terraform,
kubectl, minikube, timeout, curl, and the selected driver. Report missing
prerequisites instead of installing packages or altering host configuration
without permission.

Default to Docker on Linux, the validated parallel driver. Only stress other
drivers when explicitly requested and available; do not extrapolate Docker
results to Podman, QEMU, macOS, or Windows.

Build/install once with `make set-local`. Use the resulting
`bin/terraform-local.tfrc` via an absolute `TF_CLI_CONFIG_FILE`. All workers must
use this build, not a registry release. Avoid concurrent builds/installations.

Use a private temporary run directory (`umask 077`) with a unique run prefix.
Export a disposable `MINIKUBE_HOME` and `KUBECONFIG` beneath it. Minikube appends
`.minikube` to `MINIKUBE_HOME`. All stress profiles must share this home and
kubeconfig: separate homes would avoid the shared-file contention being tested.
Unset `TF_MINIKUBE_IN_PROCESS` and `TF_MINIKUBE_WORKER` for the normal run.
Do not replace the user's persistent environment or kubeconfig.

Caches may be seeded by COPYING compatible cache files into the disposable home;
never symlink or mutate the user's caches. Pin a Kubernetes version supported by
this provider and available to the test environment; report the exact version.

## 2. Choose maximum useful parallelism

Terraform has no unlimited `-parallelism` mode. Set it to the number of independent
cluster resources in the stress graph, rather than an arbitrary huge value.

Compute the largest host-safe cluster count `N` from actual available resources:

- Start with the smaller of CPU capacity and available-memory capacity divided
  by the per-cluster allocations. On Linux, inspect `nproc`, `MemAvailable`, and
  any tighter cgroup limits. Respect Docker VM/daemon limits where applicable.
- Reserve at least 2 CPUs and the greater of 4 GiB or 20% of available memory for
  the host, Terraform, and workers. Include additional per-worker memory overhead
  (at least 512 MiB per cluster) in the calculation; swap is not usable capacity.
- Check disk space for images, caches, and all cluster disks; lower `N` if needed.
- Use supported cluster sizes, normally 2 CPUs and 2–4 GiB per single-node Docker
  cluster. Do not disable Minikube resource checks to inflate concurrency.
- A user-specified count is an upper bound, not permission to exhaust the host.

Report the inputs, budget, chosen `N`, and limiting resource before launching.
Use `-parallelism="$N"` for apply, plan/refresh, and destroy. If `N < 2`, report
that parallel E2E is blocked by capacity; a serial pass is not a parallel pass.
Do not silently cap a capable machine at Terraform's default of 10.

## 3. Run existing E2E coverage

Run the existing black-box suite against the local build before the stress run:

```bash
export TF_CLI_CONFIG_FILE="$REPO_ROOT/bin/terraform-local.tfrc"
export TF_CLI_ARGS_apply="-parallelism=$N"
export TF_CLI_ARGS_plan="-parallelism=$N"
export TF_CLI_ARGS_destroy="-parallelism=$N"
./test/e2e/run.sh --skip-build --work-dir "$RUN_DIR/baseline" docker
```

Define `REPO_ROOT`, `RUN_DIR`, and `N` first. Capture logs and real exit codes,
using `pipefail` if piping through `tee`. The existing stack has ONE cluster and
flavours run sequentially: raising its parallelism alone does NOT test parallel
cluster creation. Do not claim otherwise. The baseline uses fixed profile names;
check for collisions in the driver before running and stop rather than touching
an existing container/profile. Do not run competing copies of that baseline.

## 4. Exercise a genuinely parallel graph

Create an ephemeral Terraform root under the run directory, using the existing
stack as a module with `count = N` (an absolute module source is simplest). Copy
the root provider requirements from `test/e2e/stack/versions.tf` so the local
provider address and version resolve correctly. Pass every required module
variable and give each instance a unique profile name derived from the run prefix
and index. No `depends_on` chains between clusters.

Alternate at least two supported CPU/memory configurations, accounting for the
larger size in the resource budget. Expose module outputs as a sensitive map;
keep state and output JSON private. Reuse the existing per-cluster kubeconfig and
health/workload checks, adapting output selection for the module map rather than
changing the tracked harness. Do not skip health checks just because apply passed.

Execute and verify:

1. `terraform init`, then `terraform apply -auto-approve -parallelism="$N"`.
2. Record timestamped worker/process activity and Docker events during apply.
   Prove at least two different profiles have overlapping provisioning intervals
   before either finishes. Container creation timestamps or total runtime alone
   are supporting evidence, not proof. If instrumentation cannot establish
   overlap, report it as unverified, not passed.
3. Inspect each saved profile under the disposable home. CPU, memory, name, and
   Kubernetes version must match that instance's requested settings.
4. Verify all expected kubeconfig entries survive concurrent writes. Build a
   separate kubeconfig per instance from its provider outputs; never rely on
   whichever shared `current-context` happened to win.
5. Run API readiness, node readiness, storage, and workload checks for EVERY
   cluster. Write a unique ConfigMap marker to each cluster and read it through
   that instance's credentials; assert the expected marker and absence of other
   clusters' markers.
6. Run `terraform plan -detailed-exitcode -parallelism="$N"`; only exit 0 is
   clean (2 means drift, 1 means error).
7. Change addons for one instance using a low-network-dependency addon supported
   by this pinned Minikube version. Apply, check the intended change and that
   other profiles/addons are unchanged, then require another clean plan.
8. Destroy with `terraform destroy -auto-approve -parallelism="$N"`. Verify empty
   state, no run-owned profiles/containers, and no surviving worker descendants.

Use explicit operation deadlines (e.g. 20 minutes for apply and 15 for destroy).
Collect peak memory, worker counts, and elapsed times. If resource pressure or an
infrastructure failure stops the run, preserve diagnostics and report the failure;
any lower-concurrency retry is a separate run, not a replacement result.

## 5. Failure containment and fallback

After the main stress run, use a small separate disposable fixture to verify:

- Refreshing a missing profile gives an actionable error, not a dead plugin;
  a subsequent operation against a surviving profile succeeds.
- Cancelling a long-running operation terminates its worker and, on Unix, its
  process-group descendants within a bounded interval. Infrastructure left by
  interrupted provisioning still needs explicit cleanup.
- A two-cluster lifecycle with `TF_MINIKUBE_IN_PROCESS=1` remains functional.
  Label this fallback coverage; do not expect provisioning overlap there.

Only inject failures into run-owned profiles/processes. Never broadly kill
provider, Terraform, Docker, or Minikube processes on the host.

## 6. Cleanup and report

Arrange cleanup before provisioning (EXIT/INT/TERM traps if scripting). Run
Terraform destroy first. For failures before state was saved, delete only the
exact run-owned profiles with the disposable environment still set. Never use
`make clean`, `make nuke`, `minikube delete --all`, or Docker prune. Keep the
disposable home until cleanup is verified; deleting it first loses recovery data.
Preserve private logs/state on failure and report any remaining resources with
exact recovery commands. Do not print client keys, sensitive outputs, or raw
Terraform state in the report or upload them as public artifacts.

Finish with a compact table: baseline, parallel create, overlap evidence,
settings isolation, per-cluster health, addon update, clean plan, parallel
destroy, failure containment, fallback, cleanup. Mark each PASS/FAIL/SKIP and
explain every skip. Include actual `N`, resource budget, versions, timings,
artifact paths, and outstanding cleanup. Never call the run fully successful
when overlap or any required assertion was not verified.
