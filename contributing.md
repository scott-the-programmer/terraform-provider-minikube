# contributing

Raising bugs, feature requests and PRs is more than welcome!

If you want to jump in and help out, here's the best way to get started

## Prerequisites

- [docker](https://www.docker.com/get-started/)
- [golang](https://go.dev/)
- [terraform 1.\* and onwards](https://www.terraform.io/)
- make
  - Windows: http://gnuwin32.sourceforge.net/packages/make.htm
  - OSX: `brew install make`
  - Debian/Ubuntu: `apt-get make`
- [minikube](https://minikube.sigs.k8s.io/docs/start/) (for testing)

## Package dependencies

```console
make init
```

## Building the binary

```console
make build
```

## Tests

### Unit Tests

```console
make test
```

### Acceptance Tests

To spin up actual clusters on your machine

```console
make acceptance
```

### End-to-End Tests

A black-box suite that has no Go in it at all: terraform stands up a cluster per
driver, `kubectl` connects to it using only the provider's outputs, nginx gets
deployed and the traffic path is verified, then everything is destroyed.

```console
make e2e              # every driver this machine can run
make e2e-docker       # just one
make e2e FLAVOURS=qemu
```

Drivers that aren't installed are reported as `SKIP` rather than failing. See
[test/e2e/README.md](./test/e2e/README.md) for the full list of assertions and
how to add a driver flavour.

## Test stack

```console
make set-local
make test-stack
```

or

```console
make set-local
make build
terraform -chdir=examples/resources/minikube_cluster apply
```

## Regenerating Schema

[schema_cluster.go](./minikube/schema_cluster.go) is generated via the `make schema` command which will use the output of your currently installed minikube version i.e. `minikube --help` to generate the terraform schema.

There are a few cases where the cli output doesn't map well to terraform. In these cases, we specify overrides in [schema_builder.go](./minikube/generator/schema_builder.go) to make sure the parameters make sense in the context of terraform

## Debugging via vscode

### Attaching to the terraform provider binary

To debug your terraform provider, run the `Debug Terraform Provider` vscode task. This will then output an environment variable that you will need to set in a new shell like so

```console
export TF_REATTACH_PROVIDERS='*output from vscode debug session'
make set-local
make test-stack
```

## Debugging via go entrypoint

You can run a self-contained cluster spin up and teardown via

```console
go run ./hack/main.go *drivername*
```

## Regenerating terradocs / mocks

Any changes to mocked interfaces and schema resources need to be
reflected in their generated counter parts

```console
make configure
```
