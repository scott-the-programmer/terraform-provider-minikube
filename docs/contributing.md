# contributing

Raising bugs, feature requests and PRs is more than welcome!

If you want to jump in and help out, here's the best way to get started

## Prerequisites

* [docker](https://www.docker.com/get-started/)
* [golang](https://go.dev/)
* [terraform 1.* and onwards](https://www.terraform.io/)
* make
  * Windows: http://gnuwin32.sourceforge.net/packages/make.htm
  * OSX: `brew install make`
  * Debian/Ubuntu: `apt-get make`
* [minikube](https://minikube.sigs.k8s.io/docs/start/) (for testing)

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

## Test stack

```console
make set-rc
make test-stack
```

or

```console
make set-rc
make build 
terraform -chdir=examples/resources/minikube_cluster apply 
```

## Debugging via vscode

### Attaching to the terraform provider binary

To debug your terraform provider, run the `Debug Terraform Provider` vscode task. This will then output an environment variable that you will need to set in a new shell like so


```console
export TF_REATTACH_PROVIDERS='*output from vscode debug session'
make set-rc
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