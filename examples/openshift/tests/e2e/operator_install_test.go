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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Operator Installation", func() {
	ctx := context.Background()

	Context("security posture", func() {
		var operatorPods []corev1.Pod

		BeforeEach(func() {
			pods, err := clientset.CoreV1().Pods(ReleaseNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: "app.kubernetes.io/name=spark-operator",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(pods.Items).NotTo(BeEmpty(), "No operator pods found in namespace %s", ReleaseNamespace)
			operatorPods = pods.Items
		})

		It("should not set fsGroup to 185 on any operator pod", func() {
			for _, pod := range operatorPods {
				if pod.Spec.SecurityContext == nil || pod.Spec.SecurityContext.FSGroup == nil {
					continue
				}
				Expect(*pod.Spec.SecurityContext.FSGroup).NotTo(Equal(int64(185)),
					fmt.Sprintf("Pod %s has fsGroup=185 (breaks OpenShift restricted-v2 SCC)", pod.Name))
			}
		})

		It("should set runAsNonRoot on all operator pods", func() {
			for _, pod := range operatorPods {
				Expect(pod.Spec.SecurityContext).NotTo(BeNil(),
					fmt.Sprintf("Pod %s has no pod-level securityContext", pod.Name))
				Expect(pod.Spec.SecurityContext.RunAsNonRoot).NotTo(BeNil(),
					fmt.Sprintf("Pod %s does not set runAsNonRoot", pod.Name))
				Expect(*pod.Spec.SecurityContext.RunAsNonRoot).To(BeTrue(),
					fmt.Sprintf("Pod %s has runAsNonRoot=false", pod.Name))
			}
		})

		It("should not run any operator container as root", func() {
			for _, pod := range operatorPods {
				for _, container := range pod.Spec.Containers {
					if container.SecurityContext == nil {
						continue
					}
					if container.SecurityContext.RunAsUser != nil {
						Expect(*container.SecurityContext.RunAsUser).NotTo(Equal(int64(0)),
							fmt.Sprintf("Container %s in pod %s has runAsUser=0", container.Name, pod.Name))
					}
					if container.SecurityContext.RunAsNonRoot != nil {
						Expect(*container.SecurityContext.RunAsNonRoot).To(BeTrue(),
							fmt.Sprintf("Container %s in pod %s has runAsNonRoot=false", container.Name, pod.Name))
					}
				}
			}
		})

		It("should enforce least-privilege on all operator containers", func() {
			for _, pod := range operatorPods {
				for _, container := range pod.Spec.Containers {
					Expect(container.SecurityContext).NotTo(BeNil(),
						fmt.Sprintf("Container %s in pod %s has no securityContext", container.Name, pod.Name))
					Expect(container.SecurityContext.AllowPrivilegeEscalation).NotTo(BeNil(),
						fmt.Sprintf("Container %s in pod %s does not set allowPrivilegeEscalation", container.Name, pod.Name))
					Expect(*container.SecurityContext.AllowPrivilegeEscalation).To(BeFalse(),
						fmt.Sprintf("Container %s in pod %s allows privilege escalation", container.Name, pod.Name))
					Expect(container.SecurityContext.ReadOnlyRootFilesystem).NotTo(BeNil(),
						fmt.Sprintf("Container %s in pod %s does not set readOnlyRootFilesystem", container.Name, pod.Name))
					Expect(*container.SecurityContext.ReadOnlyRootFilesystem).To(BeTrue(),
						fmt.Sprintf("Container %s in pod %s does not use read-only root filesystem", container.Name, pod.Name))
				}
			}
		})
	})
})
