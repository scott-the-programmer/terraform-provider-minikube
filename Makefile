.PHONY: init
init:
	go mod tidy

.PHONY: configure
configure: build set-local
	go generate ./...

.PHONY: schema
schema:
	go generate ./minikube/schema_cluster.go

.PHONY: clean
clean:
	rm bin/* || true
	rm tests/terraform.tfstate || true
	rm tests/terraform.tfstate.backup || true
	minikube delete -p terraform-provider-minikube
	minikube delete -p terraform-provider-minikube-acc

.PHONY: test
test:
	go clean -testcache
	go test ./...  -coverprofile cover.out.tmp
	cat cover.out.tmp | grep -v "mock_" > cover.out

.PHONY: acceptance
acceptance:
	go clean -testcache
	TF_ACC=true go test ./minikube -run "TestClusterCreation" -v -p 1 --timeout 10m

.PHONY: test-stack
test-stack: set-local
	terraform -chdir=examples/resources/minikube_cluster init || true
	terraform -chdir=examples/resources/minikube_cluster apply --auto-approve
	terraform -chdir=tests destroy --auto-approve

.PHONY: build
build:
	go build -o bin/terraform-provider-minikube

.PHONY: set-local 
set-local: build
	mkdir -p $$HOME/.terraform.d/plugins/registry.terraform.io/scott-the-programmer/minikube/99.99.99/linux_amd64 
	mkdir -p $$HOME/.terraform.d/plugins/registry.terraform.io/scott-the-programmer/minikube/99.99.99/darwin_amd64 
	cp bin/terraform-provider-minikube $$HOME/.terraform.d/plugins/registry.terraform.io/scott-the-programmer/minikube/99.99.99/linux_amd64/terraform-provider-minikube
	cp bin/terraform-provider-minikube $$HOME/.terraform.d/plugins/registry.terraform.io/scott-the-programmer/minikube/99.99.99/darwin_amd64/terraform-provider-minikube

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
