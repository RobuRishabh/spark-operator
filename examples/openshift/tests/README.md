# OpenShift/KIND E2E Tests - Local Development Guide

This directory contains Go/Ginkgo end-to-end tests for the Spark Operator. These tests work on both:
- **KIND clusters** (local development)
- **OpenShift clusters** (production)

The Makefile at `examples/openshift/Makefile` provides standardized make targets that can be used in GitHub Actions CI and locally on Mac/Linux.

## Overview

### What's Tested

| Test | File | What It Validates |
|------|------|-------------------|
| **Operator Install** | `suite_test.go` | Kustomize or Helm operator installation, RBAC setup |
| **Operator Security Posture** | `operator_install_test.go` | fsGroup != 185, runAsNonRoot, no root UID, least-privilege |
| **Spark Pi** | `sparkapplication_test.go` | SparkApplication CRD, driver/executor lifecycle, job completion |
| **Spark Pi (ConfigMap)** | `sparkapplication_test.go` | Volume mounts with ConfigMap |
| **Spark Pi (Custom Resource)** | `sparkapplication_test.go` | CPU/memory resource requests and limits |
| **Spark Pi (Python)** | `sparkapplication_test.go` | PySpark SparkApplication |
| **Spark Pi (Suspend/Resume)** | `sparkapplication_test.go` | Suspend and resume lifecycle |
| **Failure Cases** | `sparkapplication_test.go` | Submission failure retries, application failure retries |
| **Docling Spark** | `docling_spark_test.go` | PVC storage, multi-executor Python workload with docling image |
| **ScheduledSpark** | `scheduledsparkapplication_test.go` | ScheduledSparkApplication scheduling, finalizer cleanup |
| **SparkConnect** | `sparkconnect_test.go` | SparkConnect reconciliation |
| **Spark UI** | `spark_ui_test.go` | Spark UI service creation |
| **Prometheus Metrics** | `prometheus_metrics_test.go` | Operator metrics endpoint |

> **Note:** The docling test is labeled `Label("docling")` and excluded from default CI runs. It requires the ~9.5GB `quay.io/rishasin/docling-spark:multi-output` image to be preloaded into the cluster.

---

## Prerequisites

- **Docker** - Running and accessible
- **kubectl** - Kubernetes CLI
- **Go** (1.24+) - Required for Go e2e tests
- **kind** - For local KIND cluster setup (install via `go install sigs.k8s.io/kind` or from [kind releases](https://kind.sigs.k8s.io/docs/user/quick-start/#installation))

---

## Quick Start

> **Important:** Run all make commands from the `examples/openshift/` directory.

```bash
cd /path/to/spark-operator/examples/openshift
```

### Step 1: Build and load operator image

```bash
# From repo root: build operator, create Kind cluster, and load image
make kind-load-image IMAGE_TAG=local
```

### Step 2: Run tests

```bash
# Run all standard tests (excludes docling)
SPARK_OPERATOR_IMAGE=ghcr.io/kubeflow/spark-operator/controller:local \
  make e2e-kustomize-test
```

### Step 3: Cleanup

```bash
make kind-cleanup
```

### Optional: Build and use a local operator image (Kind only)

Build the operator image from a specific Dockerfile, load it into Kind, then run tests. Choose the architecture as needed.

```bash
# 1) Build (pick one)
# amd64:
PLATFORM=linux/amd64 make -C .. docker-build-file DOCKERFILE=examples/openshift/Dockerfile.odh IMAGE=quay.io/opendatahub/spark-operator:local
# arm64:
PLATFORM=linux/arm64 make -C .. docker-build-file DOCKERFILE=examples/openshift/Dockerfile.odh IMAGE=quay.io/opendatahub/spark-operator:local

# 2) Load into Kind
make kind-load-image-file IMAGE=quay.io/opendatahub/spark-operator:local

# 3) Run Kustomize Go e2e (uses SPARK_OPERATOR_IMAGE if set)
SPARK_OPERATOR_IMAGE=quay.io/opendatahub/spark-operator:local make e2e-kustomize-test
```

Notes:
- Tag must exactly match manifests (e.g., `quay.io/opendatahub/spark-operator:local`); `IfNotPresent` will use the preloaded node image.
- Cross-builds work on x86 via Buildx, but arm64 Kind/e2e requires an ARM64 host/runner.
- For parallel jobs, use unique tags (e.g., `:pr-<sha>`).

---

## Make Targets

| Target | Description |
|--------|-------------|
| `make kind-setup` | Setup local Kind cluster for testing |
| `make kind-cleanup` | Delete Kind cluster and cleanup resources |
| `make e2e-kustomize-test` | Run Go e2e tests with Kustomize (excludes docling by default) |
| `make e2e-docling-test` | Run docling-specific Go e2e test (requires preloaded image) |
| `make test-all` | Run all Go e2e tests including docling |
| `make docker-build-file` | Build docker image from a specific Dockerfile |
| `make kind-load-image-file` | Load a built image into Kind |

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `INSTALL_METHOD` | `helm` | Set to `kustomize` to use Kustomize manifests for operator installation |
| `SPARK_OPERATOR_IMAGE` | *(uses `params.env` default)* | Overrides the controller and webhook image in `config/default/params.env` |
| `CLEANUP` | `true` | Set to `false` to preserve resources after tests |
| `GINKGO_LABEL_FILTER` | `!docling` | Ginkgo label filter for `e2e-kustomize-test` (set empty to run all) |
| `KIND_CLUSTER_NAME` | `spark-operator` | Name of the Kind cluster |
| `K8S_VERSION` | `v1.32.0` | Kubernetes version for Kind |

### Examples

```bash
# Run all tests including docling
GINKGO_LABEL_FILTER="" make e2e-kustomize-test

# Keep resources for debugging
CLEANUP=false make e2e-kustomize-test

# Use custom operator image
SPARK_OPERATOR_IMAGE=quay.io/opendatahub/spark-operator:local make e2e-kustomize-test
```

---

## Docling Test

The docling test validates the full Spark Operator pipeline with PVC-based storage and the `quay.io/rishasin/docling-spark:multi-output` image (~9.5GB). It is excluded from default CI runs via `Label("docling")`.

### What it tests

1. Creates PVCs (`docling-input`, `docling-output`) in the test namespace
2. Uploads test PDFs from `assets/` to the input PVC via a helper pod
3. Submits the docling SparkApplication (Python, 1 driver + 2 executors)
4. Waits for completion (15-minute timeout)
5. Verifies driver logs confirm successful processing
6. Verifies executor pods were created

### Running locally

```bash
# 1. Setup Kind cluster
make kind-setup

# 2. Preload the docling-spark image (~9.5GB)
docker pull quay.io/rishasin/docling-spark:multi-output
kind load docker-image quay.io/rishasin/docling-spark:multi-output --name spark-operator

# 3. Build and load operator image (from repo root)
make -C ../.. kind-load-image IMAGE_TAG=local

# 4. Run docling test
SPARK_OPERATOR_IMAGE=ghcr.io/kubeflow/spark-operator/controller:local \
  make e2e-docling-test
```

---

## Test Suite Architecture

The test suite in `e2e/suite_test.go` supports a toggle via the `INSTALL_METHOD` environment variable:

- `INSTALL_METHOD=helm` (default) — installs the operator using the Helm chart (same as `test/e2e/`)
- `INSTALL_METHOD=kustomize` — installs the operator using `kubectl apply -k config/default/ --server-side=true`

When using Kustomize mode, the test also:
- Overrides the operator image in `config/default/params.env` if `SPARK_OPERATOR_IMAGE` is set
- Applies Spark driver ServiceAccount and RBAC via `config/spark-rbac/`

### Running step by step

```bash
# Build and load image into Kind
make kind-load-image IMAGE_TAG=local

# Run from repo root with environment variables
INSTALL_METHOD=kustomize \
  SPARK_OPERATOR_IMAGE=ghcr.io/kubeflow/spark-operator/controller:local \
  go test ./examples/openshift/tests/e2e/ -v -ginkgo.v -timeout 30m
```

### CI integration

The `kustomize-e2e-test` job in `.github/workflows/kustomize-e2e.yaml` builds the operator image from the PR, loads it into Kind, and runs these tests with `SPARK_OPERATOR_IMAGE` set to the locally built image. It triggers on PRs that touch `config/`, `examples/openshift/tests/e2e/`, operator source code, or the workflow file itself, and runs across the same Kubernetes version matrix as the upstream Helm-based e2e tests.

---

## Architecture

```
┌───────────────────────────────────────────────────────────┐
│                      Kind Cluster                         │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              spark-operator namespace               │  │
│  │  ┌─────────────────┐  ┌─────────────────────────┐   │  │
│  │  │   Controller    │  │       Webhook           │   │  │
│  │  │      Pod        │  │         Pod             │   │  │
│  │  └─────────────────┘  └─────────────────────────┘   │  │
│  └─────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                spark-test namespace                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │   Driver    │  │  Executor   │  │    PVCs     │  │  │
│  │  │    Pod      │  │    Pods     │  │ input/output│  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

---

## Files in This Directory

| File | Purpose |
|------|---------|
| `setup-kind-cluster.sh` | Creates Kind cluster and prerequisites |
| `cleanup-kind-cluster.sh` | Deletes Kind cluster and resources |
| `assets/` | Test PDF files for docling tests |
| `e2e/suite_test.go` | Test suite setup with Helm/Kustomize toggle |
| `e2e/operator_install_test.go` | Operator security posture checks (fsGroup, runAsNonRoot, least-privilege) |
| `e2e/sparkapplication_test.go` | SparkApplication e2e specs (spark-pi, failures, suspend/resume) |
| `e2e/docling_spark_test.go` | Docling SparkApplication with PVC storage |
| `e2e/scheduledsparkapplication_test.go` | ScheduledSparkApplication specs |
| `e2e/sparkconnect_test.go` | SparkConnect reconciliation spec |
| `e2e/sparkconnect_query_test.go` | SparkConnect query execution spec |
| `e2e/prometheus_metrics_test.go` | Prometheus metrics spec |
| `e2e/spark_ui_test.go` | Spark UI spec |
| `e2e/examples/` | YAML fixtures for test SparkApplications |
| `e2e/bad_examples/` | YAML fixtures for failure/retry test cases |
