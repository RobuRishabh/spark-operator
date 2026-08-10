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

package e2e_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeflow/spark-operator/v2/api/v1beta2"
)

// These tests prove the validating webhook rejects bad SparkApplications at admission time.
// They run under INSTALL_METHOD=preinstalled (module CI) as well as helm/kustomize installs.
var _ = Describe("Validating webhook admission", func() {
	Context("invalid SparkApplication", func() {
		ctx := context.Background()
		mainFile := "local:///opt/spark/examples/jars/spark-examples.jar"

		It("should reject a SparkApplication with conflicting node selectors", func() {
			app := &v1beta2.SparkApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "webhook-reject-nodeselector",
					Namespace: TestNamespace,
				},
				Spec: v1beta2.SparkApplicationSpec{
					Type:                v1beta2.SparkApplicationTypeScala,
					SparkVersion:        "4.0.1",
					Mode:                v1beta2.DeployModeCluster,
					MainApplicationFile: &mainFile,
					MainClass:           ptr.To("org.apache.spark.examples.SparkPi"),
					NodeSelector: map[string]string{
						"role": "shared",
					},
					Driver: v1beta2.DriverSpec{
						SparkPodSpec: v1beta2.SparkPodSpec{
							Cores:  ptr.To[int32](1),
							Memory: ptr.To("512m"),
							NodeSelector: map[string]string{
								"role": "driver",
							},
						},
					},
					Executor: v1beta2.ExecutorSpec{
						SparkPodSpec: v1beta2.SparkPodSpec{
							Cores:  ptr.To[int32](1),
							Memory: ptr.To("512m"),
						},
						Instances: ptr.To[int32](1),
					},
				},
			}

			By("Creating SparkApplication that violates validating webhook rules")
			err := k8sClient.Create(ctx, app)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("node selector cannot be defined"))
		})

		It("should accept a valid SparkApplication via dry-run", func() {
			app := &v1beta2.SparkApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "webhook-accept-dry-run",
					Namespace: TestNamespace,
				},
				Spec: v1beta2.SparkApplicationSpec{
					Type:                v1beta2.SparkApplicationTypeScala,
					SparkVersion:        "4.0.1",
					Mode:                v1beta2.DeployModeCluster,
					MainApplicationFile: &mainFile,
					MainClass:           ptr.To("org.apache.spark.examples.SparkPi"),
					Driver: v1beta2.DriverSpec{
						SparkPodSpec: v1beta2.SparkPodSpec{
							Cores:  ptr.To[int32](1),
							Memory: ptr.To("512m"),
						},
					},
					Executor: v1beta2.ExecutorSpec{
						SparkPodSpec: v1beta2.SparkPodSpec{
							Cores:  ptr.To[int32](1),
							Memory: ptr.To("512m"),
						},
						Instances: ptr.To[int32](1),
					},
				},
			}

			By("Dry-running SparkApplication create through the validating webhook")
			Expect(k8sClient.Create(ctx, app, client.DryRunAll)).To(Succeed())
		})
	})
})
