# Parallel cluster creation via worker process isolation

Issue [#285](https://github.com/scott-the-programmer/terraform-provider-minikube/issues/285) handoff:
this document describes the implemented process-isolation design, its coordination model,
fallbacks, and the evidence collected on this host.

## The problem

Minikube's `node.Start`/`node.Provision` paths persist cluster configuration through
process-global Viper singletons (profile name, HA, bootstrapper, preload), and additional-node
paths read the same singleton. Running two cluster operations inside one provider process can
therefore save cluster A's configuration under cluster B's profile. The provider used to guard
against this with one broad creation mutex, serializing *all* cluster operations on the host,
regardless of profile.

A memory-ordering fix cannot close this: with ordered access, A-set → B-set → A-read still
observes B's settings (demonstrated in the issue investigation with `go run -race`).

## The design

Each lifecycle operation re-executes the provider binary in a dedicated worker process
(`os.Executable()`), selected before Terraform plugin serving in `main.go` through
`TF_MINIKUBE_WORKER`. The worker:

1. runs one request from stdin (a typed JSON document; no credentials or state transit through
   argv), emits exactly one marker-framed JSON response on stdout, and logs to stderr;
2. reconstructs the operation's complete configuration from the request (nothing is inherited
   from any other operation);
3. calls the same embedded Minikube integration (`MinikubeClient.Start`, addon helpers,
   profile deletes) pinned in `go.mod`; no external binary, no upstream fork;
4. exits, so its process-global Minikube state dies with it.

Terraform state handling stays in the parent. Creation returns the serializable subset of
`kubeconfig.Settings` the resource schema consumes, and refresh returns the saved
`config.ClusterConfig`; meanings of existing attributes and outputs are unchanged.

Operations stay on fresh workers per request; no long-lived worker is reused across profiles.
Upstream `exit.Error` (which calls `os.Exit`) and panics can no longer take the plugin down:
the parent classifies a child exit — with its stderr tail attached — as a typed protocol
failure. Package initialization (PATH pre-pending, log registration) still happens before
dispatch and is inert for workers; worker `Serve` re-binds the Minikube logging bridges to
stderr so logs never collide with the response frame or Terraform's own gRPC stdio.

## Coordination

* *Same profile across relevant operations*: the parent serializes on a keyed
  (minikube home, profile) lock. Distinct profiles overlap.
* *Shared host resources*: Minikube's own file locks still protect kubeconfig
  read/modify/write, certificate setup, and the download caches; workers inherit them.
* The broad creation mutex only applies to the documented fallback path (below).

The refresh path also runs in a worker on purpose: `mustload.Partial` terminates its process
when a profile is missing, which in-process kills the entire Terraform plugin. Through a
worker this becomes a contained, actionable error.

## Cancellation and failure

Terraform context cancellation propagates as a kill of the whole worker process group
(`Setpgid` on POSIX; a direct kill otherwise), so driver-spawned descendants do not outlive
the operation. A propagated wall-clock deadline rides inside the request. Workers are always
reaped before `Run` returns. Malformed output, abnormal exit codes, and worker death all
surface with the worker's stderr tail so the practitioner can act (retry create, or run
`minikube delete -p <profile>`); the provider never silently retries an uncertain create.

## Opting out

Set `TF_MINIKUBE_IN_PROCESS=1` to run operations in the plugin process again — the previous
behavior, with the broad creation mutex retaining its serialized safety for drivers whose
parallel safety has not been established.

## What was verified here (Docker driver, Linux host, 32 CPU / 124 GiB)

With `MINIKUBE_HOME` pointed at a disposable home seeded by *copying* (never mutating) the
user's preexisting Minikube caches, pinned to `v1.30.2` matching those warm caches:

* **Parallel creation**: one `terraform apply -parallelism=2` with two independent docker
  cluster resources. Both KIC containers were created inside the same two-second window
  (`docker inspect` `CreatedAt` per profile; a 1-second sampler watching the Minikube labels
  showed `alpha=1 beta=1` throughout the creation window). Cold-create wall time for the pair
  was ~24 s, versus ~22 s for a single cold cluster created serially — the pair completed in
  about the time of *one* serial creation (a serialized mutex path would cost ~2×).
* **Settings isolation at the boundary**: `probe-isolation` workers set Viper profile globals
  concurrently like a start does, cross-process; each reports its own value (`go test -race`
  in `minikube/worker`). On the real path, saved profile configs contained each cluster's own
  size and endpoints (alpha 2 CPU/4096 MB, beta 1 CPU/2048 MB), kubeconfig gained both
  entries, and each cluster accepted its own credentials.
* **Addon updates and refresh**: addon changes wrote through a worker and a second
  `terraform plan` reported no drift; unmentioned addons stayed untouched.
* **Concurrent delete across profiles**: `terraform destroy -parallelism=2` removed both
  clusters inside one 12 s window with no cross-profile effects.
* **Failure handling**: cache miss surfaced `worker exit status 50` with the upstream
  stderr tail; refresh of a deleted profile surfaced `profile "x" not found in minikube home
  ...` instead of terminating the provider process.

**current-context under parallelism**: both clusters' entries survive; the documented
behavior is that `current-context` ends up pointing at whichever cluster registered last,
so per-resource outputs (the `host`, `client_key`, ... attributes) remain the authoritative
access path for parallel stacks. Copying the user's caches into `MINIKUBE_HOME` for the test
meant the user's own caches were never deleted or modified.

## Known limitations

* Verified only for the **docker/KIC driver on Linux**. Other drivers retain the documented
  fallback (`TF_MINIKUBE_IN_PROCESS=1`) until their shared-resource behavior (networks,
  subnets, ports, certificate setup) is audited; claims of parallel safety are limited to the
  tested driver.
* The `registry`/`metrics-server` addon families pull pod images from inside the cluster;
  on network-restricted hosts their waits can exceed the addon deadline (observed here and
  unrelated to parallelism).
* Docker-machine reused-but-half-provisioned profiles (e.g. leftover containers and KIC
  volumes after a failed run) reproduce the pre-existing upstream reuse fragility. `terraform
  destroy` after a clean state, or `minikube delete -p <profile>`, is the recovery path.
* Force-kill still relies on the driver's own reaping on Windows (process groups are POSIX).

## Test seams

Worker tests live in `minikube/worker`. `TF_MINIKUBE_WORKER_TEST_*` variables only take
effect inside worker processes (and the re-executed test binary) and are documented in
`protocol.go`.
