.SILENT:

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Directory paths (relative to this Makefile in examples/openshift/)
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
REPO_ROOT := $(abspath $(MAKEFILE_DIR)/../..)
TESTS_DIR := $(MAKEFILE_DIR)/tests
CONTEXT ?= $(REPO_ROOT)

# Include repo root bin/ in PATH so tools installed by the root Makefile
# (e.g. kind via 'make kind') are available to the shell scripts.
LOCALBIN := $(REPO_ROOT)/bin
export PATH := $(LOCALBIN):$(PATH)

# Configuration
KIND_CLUSTER_NAME ?= spark-operator

# Ginkgo label filter: excludes the heavy docling test by default.
# Override with GINKGO_LABEL_FILTER="" to include it.
GINKGO_LABEL_FILTER ?= !docling

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Cluster Setup

.PHONY: kind-setup
kind-setup: ## Setup a local Kind cluster for testing.
	@echo "Setting up Kind cluster..."
	chmod +x $(TESTS_DIR)/setup-kind-cluster.sh
	$(TESTS_DIR)/setup-kind-cluster.sh

.PHONY: kind-cleanup
kind-cleanup: ## Delete the Kind cluster and cleanup resources.
	@echo "Cleaning up Kind cluster..."
	chmod +x $(TESTS_DIR)/cleanup-kind-cluster.sh
	$(TESTS_DIR)/cleanup-kind-cluster.sh

##@ Tests
## All tests use Go/Ginkgo. The docling test is excluded by default
## (requires ~9.5GB image preloaded). Override with GINKGO_LABEL_FILTER="".

.PHONY: e2e-kustomize-test
e2e-kustomize-test: ## Run Go e2e tests using Kustomize manifests (excludes docling by default).
	@echo "Running Go e2e tests with Kustomize installation..."
	cd $(REPO_ROOT) && INSTALL_METHOD=kustomize go test ./examples/openshift/tests/e2e/ -v -ginkgo.v -ginkgo.label-filter="$(GINKGO_LABEL_FILTER)" -timeout 30m -coverprofile cover-e2e-kustomize.out

.PHONY: e2e-docling-test
e2e-docling-test: ## Run docling-specific Go e2e test (requires preloaded 9.5GB image).
	@echo "Running docling Go e2e test..."
	cd $(REPO_ROOT) && INSTALL_METHOD=kustomize go test ./examples/openshift/tests/e2e/ -v -ginkgo.v -ginkgo.label-filter="docling" -timeout 45m -coverprofile cover-e2e-docling.out

.PHONY: test-all
test-all: ## Run all Go e2e tests including docling (requires preloaded docling image).
	@echo "Running all Go e2e tests..."
	cd $(REPO_ROOT) && INSTALL_METHOD=kustomize go test ./examples/openshift/tests/e2e/ -v -ginkgo.v -timeout 45m -coverprofile cover-e2e-all.out

##@ Image Build/Load (CI and local)

.PHONY: docker-build-file
docker-build-file: ## Build docker image with a specific Dockerfile. Usage: make -C examples/openshift docker-build-file DOCKERFILE=path IMAGE=<full_image>
	@if [ -z "$(DOCKERFILE)" ]; then echo "ERROR: DOCKERFILE is required (e.g., DOCKERFILE=examples/openshift/Dockerfile.odh)"; exit 1; fi
	@if [ -z "$(IMAGE)" ]; then echo "ERROR: IMAGE is required (e.g., IMAGE=quay.io/opendatahub/spark-operator:local)"; exit 1; fi
	@DF="$(DOCKERFILE)"; \
	if [[ "$$DF" != /* ]]; then DF="$(REPO_ROOT)/$$DF"; fi; \
	if [ -n "$(PLATFORM)" ]; then \
		echo "Building for platform $(PLATFORM)"; \
		docker build --platform $(PLATFORM) -f "$$DF" -t ${IMAGE} $(CONTEXT) ; \
	else \
		docker build -f "$$DF" -t ${IMAGE} $(CONTEXT) ; \
	fi

.PHONY: kind-load-image-file
kind-load-image-file: ## Load a specifically built image into Kind. Usage: make -C examples/openshift kind-load-image-file IMAGE=<full_image>
	@if [ -z "$(IMAGE)" ]; then echo "ERROR: IMAGE is required (e.g., IMAGE=quay.io/opendatahub/spark-operator:local)"; exit 1; fi
	kind load docker-image --name $(KIND_CLUSTER_NAME) $(IMAGE)
