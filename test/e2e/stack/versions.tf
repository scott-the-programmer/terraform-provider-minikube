terraform {
  required_version = ">= 1.0"

  required_providers {
    minikube = {
      source = "scott-the-programmer/minikube"
      # 99.99.99 is the version `make set-local` publishes into the local
      # filesystem mirror. run.sh rewrites this line when E2E_PROVIDER_VERSION
      # points at a released version instead.
      version = "99.99.99"
    }
  }
}
