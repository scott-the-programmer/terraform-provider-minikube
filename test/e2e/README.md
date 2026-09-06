# End-to-end tests

Black-box tests for `terraform-provider-minikube`. They stand a real cluster up
with terraform, connect to it with `kubectl` using nothing but the provider's
own outputs, deploy nginx, prove the traffic path works, and tear it all down
again.

There is no Go in here. The suite is bash + `terraform` + `kubectl`, so it
exercises the provider the way a user does — through the published resource
schema and its outputs — rather than through the provider's own test harness.

## Layout

```
test/e2e/
├── run.sh              entry point
├── stack/              one driver-agnostic terraform configuration
├── flavours/           per-driver variables (*.tfvars) and metadata (*.env)
├── manifests/nginx.yaml the workload under test
└── lib/                logging, assertions, terraform and kubectl helpers
```

Each flavour gets its own copy of `stack/` under `.work/<flavour>/`, so state,
plugin cache and lock files never collide between drivers run back to back.

## Requirements

On the host: `terraform`, `kubectl`, `curl`, `base64`, `timeout`, plus `make`
and the Go toolchain if you want `run.sh` to build the provider for you.

Per flavour, whatever that driver needs:

| Flavour  | Driver   | Needs                                       |
| -------- | -------- | ------------------------------------------- |
| `docker` | `docker` | a working docker daemon                     |
| `qemu`   | `qemu2`  | `qemu-system-<arch>`                        |
| `podman` | `podman` | a working podman, runs the cluster on cri-o |

A flavour whose driver is not installed is reported as `SKIP`, not `FAIL`, so
the suite is useful on a machine that only has one of them.

## Running

```sh
./test/e2e/run.sh                      # every flavour this machine can run
./test/e2e/run.sh docker               # one flavour
./test/e2e/run.sh docker qemu          # several
./test/e2e/run.sh --keep docker        # leave the cluster up to poke at
./test/e2e/run.sh --skip-build docker  # reuse the provider already installed
./test/e2e/run.sh --preload-image ...  # side-load nginx instead of pulling it
./test/e2e/run.sh --list               # what's available
```

Or through the makefile:

```sh
make e2e                  # every flavour
make e2e-docker           # just docker
make e2e FLAVOURS=qemu    # pick your own
```

By default `run.sh` runs `make set-local` first, which builds the provider and
publishes it as version `99.99.99` into the local filesystem mirror, then points
terraform at that mirror. To test a published build instead:

```sh
E2E_PROVIDER_VERSION=1.2.3 ./test/e2e/run.sh docker
```

The workload image is pulled by the cluster itself, which is part of what the
suite proves. Where that is not possible — offline, behind a Docker Hub rate
limit, or on a host whose DNS the cluster cannot use — `--preload-image` (or
`E2E_PRELOAD_IMAGE=1`) pulls the image on the host instead and side-loads it
into the cluster with `minikube image load`.

`--keep` leaves the clusters and their terraform directories in place; the
script prints the `terraform destroy` command for each. Everything else — an
interrupt, a failed apply, a failing check — still tears the cluster down,
because a leftover minikube profile wedges the next run.

## What is asserted

Provider surface:

- `terraform apply` creates the cluster for the driver under test
- `host`, `cluster_ca_certificate`, `client_certificate` and `client_key`
  outputs are populated and well formed
- a re-`plan` after apply is clean — the provider reads back everything it wrote

Cluster:

- a kubeconfig built purely from those outputs authenticates against the
  apiserver
- every node reports `Ready`
- `kube-system` pods are ready
- the `storage-provisioner` addon produced a default storage class

Workload:

- the nginx manifests apply and the deployment rolls out
- the expected number of pods are `Running` and the service has that many ready
  endpoints
- cluster DNS resolves `nginx.e2e.svc.cluster.local` to the service ClusterIP
- pod → service → pod HTTP returns the sentinel body
- host → apiserver → pod works via `kubectl port-forward`
- host → node IP → NodePort works, where the driver's network is routable from
  the host (skipped with a reason where it is not, e.g. qemu user networking)
- scaling to 3 replicas updates the endpoints
- the workload deletes cleanly

Teardown:

- `terraform destroy` empties the state and the apiserver stops answering

The nginx pod serves a known sentinel string from a ConfigMap, so a check can
only pass by reaching *that* nginx — not something else answering on port 80.
The image is `nginx:1.27-alpine` because busybox `wget` and `nslookup` ship in
it, which keeps the in-cluster checks from needing a second image pull.

## Adding a flavour

Drop a `flavours/<name>.tfvars` with the variables from `stack/variables.tf`,
and a `flavours/<name>.env` describing how to detect the driver:

```sh
E2E_DESCRIPTION="kvm2 driver (libvirt VM)"
E2E_REQUIRES="virsh"          # space separated; "a|b" means either will do
E2E_HEALTHCHECK="virsh list"  # optional, catches installed-but-not-working
E2E_APPLY_TIMEOUT=1800
```

No changes to `run.sh` are needed — it discovers flavours from the directory.
