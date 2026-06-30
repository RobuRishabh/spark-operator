/*
Copyright 2024 The Kubeflow authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kustomize_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

// TestOverlayBuilds validates that both ODH and RHOAI overlays build
// successfully and produce expected core resources. This catches regressions
// like broken patch targets after upstream renames or invalid kustomization references.
func TestOverlayBuilds(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	overlays := []struct {
		name      string
		path      string
		namespace string
	}{
		{"odh", filepath.Join(repoRoot, "config", "overlays", "odh"), "opendatahub"},
		{"rhoai", filepath.Join(repoRoot, "config", "overlays", "rhoai"), "redhat-ods-applications"},
	}

	for _, overlay := range overlays {
		t.Run(overlay.name, func(t *testing.T) {
			resources := buildKustomizePath(t, overlay.path)

			t.Run("CoreResources", func(t *testing.T) {
				controllerDep := findResource(resources, "Deployment", "spark-operator-controller")
				require.NotNil(t, controllerDep, "Deployment/spark-operator-controller not found in %s overlay", overlay.name)

				webhookDep := findResource(resources, "Deployment", "spark-operator-webhook")
				require.NotNil(t, webhookDep, "Deployment/spark-operator-webhook not found in %s overlay", overlay.name)

				podMonitor := findResource(resources, "PodMonitor", "spark-operator-podmonitor")
				require.NotNil(t, podMonitor, "PodMonitor/spark-operator-podmonitor not found in %s overlay", overlay.name)
			})

			t.Run("NamespaceOverride", func(t *testing.T) {
				controllerDep := findResource(resources, "Deployment", "spark-operator-controller")
				require.NotNil(t, controllerDep)
				assert.Equal(t, overlay.namespace, controllerDep.GetNamespace(),
					"Deployment namespace should be overridden to %s", overlay.namespace)
			})

			t.Run("NoNamespaceResource", func(t *testing.T) {
				ns := findResource(resources, "Namespace", "spark-operator")
				assert.Nil(t, ns, "Namespace/spark-operator should be deleted by overlay (managed externally)")
			})

			t.Run("ImageReplacement", func(t *testing.T) {
				for _, depName := range []string{"spark-operator-controller", "spark-operator-webhook"} {
					depObj := findResource(resources, "Deployment", depName)
					require.NotNil(t, depObj)
					dep := convertTo[appsv1.Deployment](t, depObj)
					for _, c := range dep.Spec.Template.Spec.Containers {
						assert.NotEmpty(t, c.Image, "container image should not be empty in %s/%s", depName, c.Name)
						assert.NotContains(t, c.Image, "ghcr.io/kubeflow",
							"overlay %s should override upstream image in %s container %s", overlay.name, depName, c.Name)
					}
				}
			})

			t.Run("NetworkPolicy", func(t *testing.T) {
				nps := findResources(resources, "NetworkPolicy")
				assert.NotEmpty(t, nps, "overlay %s should include NetworkPolicy", overlay.name)
			})
		})
	}
}
