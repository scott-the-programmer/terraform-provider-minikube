.PHONY: init
init:
	go mod tidy

.PHONY: configure
configure: build set-local
	go generate ./...

.PHONY: schema
schema:
	go generate ./minikube/schema_cluster.go
	go fmt ./minikube/schema_cluster.go

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: clean
clean:
	rm bin/* || true
	rm examples/resources/minikube_cluster/terraform.tfstate || true
	rm examples/resources/minikube_cluster/terraform.tfstate.backup || true
	rm examples/resources/minikube_cluster/.terraform.lock.hcl || true
	rm -rf examples/resources/minikube_cluster/.terraform || true
	minikube delete -p terraform-provider-minikube --purge
	minikube delete -p terraform-provider-minikube-acc --purge
	minikube delete -p terraform-provider-minikube-acc-docker --purge
	minikube delete -p terraform-provider-minikube-acc-hyperkit --purge
	minikube delete -p terraform-provider-minikube-acc-hyperv --purge

.PHONY: nuke
nuke: clean
	rm -rf ~/.minikube || true

.PHONY: test
test:
	go clean -testcache
	go test ./...  -coverprofile cover.out.tmp
	cat cover.out.tmp | grep -v "mock_" > cover.out

.PHONY: acceptance
acceptance:
	go clean -testcache
	go test -c -ldflags="-X k8s.io/minikube/pkg/version.storageProvisionerVersion=v5" -o testBinary ./minikube 
	TF_ACC=true ./testBinary -test.run "TestClusterCreation" -test.v -test.parallel 1 -test.timeout 20m

.PHONY: test-stack-apply
test-stack-apply: set-local
	terraform -chdir=examples/resources/minikube_cluster init || true
	terraform -chdir=examples/resources/minikube_cluster apply --auto-approve

.PHONY: test-stack-delete
test-stack-delete:
	terraform -chdir=examples/resources/minikube_cluster destroy --auto-approve

.PHONY: test-stack
test-stack: test-stack-apply test-stack-delete

STORAGE_PROVISIONER_TAG ?= v5
.PHONY: build
build:
	go build -o bin/terraform-provider-minikube -ldflags="-X k8s.io/minikube/pkg/version.storageProvisionerVersion=$(STORAGE_PROVISIONER_TAG)"

ARCH_RAW := $(shell uname -m)
ifeq ($(ARCH_RAW), x86_64)
	ARCH := amd64
else ifeq ($(ARCH_RAW), aarch64)
	ARCH := arm64
else
	ARCH := $(ARCH_RAW)
endif

OS_NAME := $(shell uname -s | tr A-Z a-z)
PLUGIN_NAME := terraform-provider-minikube
VERSION := 99.99.99
DEST_DIR := $$HOME/.terraform.d/plugins/registry.terraform.io/scott-the-programmer/minikube/$(VERSION)
EXT :=

ifeq ($(OS), Windows_NT)
	OS_NAME := windows
	DEST_DIR := $$APPDATA/terraform.d/plugins/registry.terraform.io/scott-the-programmer/minikube/$(VERSION)
	EXT := .exe
endif

.PHONY: set-local
set-local: build
	mkdir -p $(DEST_DIR)/$(OS_NAME)_$(ARCH) && \
	cp bin/$(PLUGIN_NAME) $(DEST_DIR)/$(OS_NAME)_$(ARCH)/$(PLUGIN_NAME)$(EXT)

.PHONY: reset-local
reset-local:
	rm -rf $(DEST_DIR)/$(OS_NAME)_$(ARCH)/$(PLUGIN_NAME)$(EXT)


SED_FLAGS := -i
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
		SED_FLAGS += -e 
endif
ifeq ($(UNAME_S),Darwin)
		SED_FLAGS += ''
endif
.PHONY: set-version
set-version:
	$(eval VERSION := $(shell cat minikube/version/version.go | grep Version | tr -d "[:space:]" | sed 's/Version\="//g' | sed 's/"\/\/.*//g'))
	sed $(SED_FLAGS) 's/VERSION=".*"/VERSION="$(VERSION)"/g' bootstrap/install-driver.sh
