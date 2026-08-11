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

# Containerized schema generation with version control
# Usage: make schema-container
#        make schema-container MINIKUBE_VERSION=v1.36.0
MINIKUBE_VERSION ?= v1.38.0
.PHONY: schema-container
schema-container:
	./scripts/schema-container.sh $(MINIKUBE_VERSION)

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
	go test -tags $(BUILD_TAGS) ./...  -coverprofile cover.out.tmp
	cat cover.out.tmp | grep -v "mock_" > cover.out

.PHONY: acceptance
acceptance:
	go clean -testcache
	go test -c -tags $(BUILD_TAGS) -ldflags="-X k8s.io/minikube/pkg/version.storageProvisionerVersion=v5" -o testBinary ./minikube 
	TF_ACC=true ./testBinary -test.run "TestClusterCreation" -test.v -test.parallel 1 -test.timeout 20m

TEST_STACK_DIR := examples/resources/minikube_cluster
LOCAL_CLI_CONFIG := $(CURDIR)/bin/terraform-local.tfrc

.PHONY: test-stack-apply
test-stack-apply: set-local
	# A rebuilt local provider has a new checksum, so regenerate the ignored lock file.
	rm -f $(TEST_STACK_DIR)/.terraform.lock.hcl
	TF_CLI_CONFIG_FILE="$(LOCAL_CLI_CONFIG)" terraform -chdir=$(TEST_STACK_DIR) init
	TF_CLI_CONFIG_FILE="$(LOCAL_CLI_CONFIG)" terraform -chdir=$(TEST_STACK_DIR) apply --auto-approve

.PHONY: test-stack-delete
test-stack-delete: local-cli-config
	TF_CLI_CONFIG_FILE="$(LOCAL_CLI_CONFIG)" terraform -chdir=$(TEST_STACK_DIR) destroy --auto-approve

.PHONY: test-stack
test-stack: test-stack-apply test-stack-delete

STORAGE_PROVISIONER_TAG ?= v5
BUILD_TAGS ?= libvirt_dlopen
.PHONY: build
build:
	go build -tags $(BUILD_TAGS) -o bin/terraform-provider-minikube -ldflags="-X k8s.io/minikube/pkg/version.storageProvisionerVersion=$(STORAGE_PROVISIONER_TAG)"

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
PLUGIN_MIRROR_DIR := $$HOME/.terraform.d/plugins
DEST_DIR := $(PLUGIN_MIRROR_DIR)/registry.terraform.io/scott-the-programmer/minikube/$(VERSION)
EXT :=

ifeq ($(OS), Windows_NT)
	OS_NAME := windows
	PLUGIN_MIRROR_DIR := $$APPDATA/terraform.d/plugins
	DEST_DIR := $(PLUGIN_MIRROR_DIR)/registry.terraform.io/scott-the-programmer/minikube/$(VERSION)
	EXT := .exe
endif

.PHONY: local-cli-config
local-cli-config:
	mkdir -p bin
	printf '%s\n' \
		'provider_installation {' \
		'  filesystem_mirror {' \
		"    path = \"$(PLUGIN_MIRROR_DIR)\"" \
		'    include = ["scott-the-programmer/minikube"]' \
		'  }' \
		'  direct {' \
		'    exclude = ["scott-the-programmer/minikube"]' \
		'  }' \
		'}' > "$(LOCAL_CLI_CONFIG)"

.PHONY: set-local
set-local: build local-cli-config
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
