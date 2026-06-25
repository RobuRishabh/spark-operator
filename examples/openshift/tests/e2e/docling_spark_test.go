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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/kubeflow/spark-operator/v2/api/v1beta2"
	"github.com/kubeflow/spark-operator/v2/pkg/util"
)

const (
	doclingWaitTimeout  = 15 * time.Minute
	doclingPollInterval = 5 * time.Second

	inputPVCName  = "docling-input"
	outputPVCName = "docling-output"
	helperPodName = "pvc-seeder"
	helperImage   = "busybox:1.37"
)

var _ = Describe("Docling SparkApplication", Label("docling"), func() {
	Context("docling-spark-job", func() {
		ctx := context.Background()
		path := filepath.Join("examples", "docling-spark-app.yaml")
		app := &v1beta2.SparkApplication{}

		BeforeEach(func() {
			By("Creating input PVC")
			Expect(k8sClient.Create(ctx, newPVC(inputPVCName, TestNamespace, "1Gi"))).To(Succeed())

			By("Creating output PVC")
			Expect(k8sClient.Create(ctx, newPVC(outputPVCName, TestNamespace, "1Gi"))).To(Succeed())

			By("Uploading test PDFs to input PVC")
			seedPVCWithTestAssets(ctx, TestNamespace, inputPVCName)

			By("Parsing SparkApplication from file")
			file, err := os.Open(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(file).NotTo(BeNil())

			decoder := yaml.NewYAMLOrJSONDecoder(file, 100)
			Expect(decoder).NotTo(BeNil())
			Expect(decoder.Decode(app)).NotTo(HaveOccurred())

			if app.Namespace == "" {
				app.Namespace = TestNamespace
			}

			By("Creating SparkApplication")
			Expect(k8sClient.Create(ctx, app)).To(Succeed())
		})

		AfterEach(func() {
			if strings.EqualFold(os.Getenv("CLEANUP"), "false") && CurrentSpecReport().Failed() {
				return
			}

			By("Deleting SparkApplication")
			key := types.NamespacedName{Namespace: app.Namespace, Name: app.Name}
			if err := k8sClient.Get(ctx, key, app); err == nil {
				Expect(k8sClient.Delete(ctx, app)).To(Succeed())
			}

			By("Deleting helper pod")
			_ = k8sClient.Delete(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: helperPodName, Namespace: TestNamespace},
			})

			By("Deleting PVCs")
			_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: inputPVCName, Namespace: TestNamespace},
			})
			_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: outputPVCName, Namespace: TestNamespace},
			})
		})

		It("should complete successfully with PVC storage", func() {
			By("Waiting for SparkApplication to complete")
			key := types.NamespacedName{Namespace: app.Namespace, Name: app.Name}
			Expect(waitForDoclingCompleted(ctx, key)).NotTo(HaveOccurred())

			By("Checking driver logs for successful processing")
			driverPodName := util.GetDriverPodName(app)
			bytes, err := clientset.CoreV1().Pods(app.Namespace).GetLogs(driverPodName, &corev1.PodLogOptions{}).Do(ctx).Raw()
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes).NotTo(BeEmpty())
			Expect(string(bytes)).To(ContainSubstring("ALL DONE"))

			By("Verifying executor pods were created")
			pods, err := clientset.CoreV1().Pods(app.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("spark-role=executor,spark-app-name=%s", app.Name),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pods.Items)).To(BeNumerically(">=", 1))
		})
	})
})

func newPVC(name, namespace, size string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}
}

func waitForDoclingCompleted(ctx context.Context, key types.NamespacedName) error {
	cancelCtx, cancelFunc := context.WithTimeout(ctx, doclingWaitTimeout)
	defer cancelFunc()

	app := &v1beta2.SparkApplication{}
	return wait.PollUntilContextCancel(cancelCtx, doclingPollInterval, true, func(ctx context.Context) (bool, error) {
		if err := k8sClient.Get(ctx, key, app); err != nil {
			return false, err
		}
		switch app.Status.AppState.State {
		case v1beta2.ApplicationStateFailedSubmission, v1beta2.ApplicationStateFailed:
			return false, errors.New(app.Status.AppState.ErrorMessage)
		case v1beta2.ApplicationStateCompleted:
			return true, nil
		}
		return false, nil
	})
}

func seedPVCWithTestAssets(ctx context.Context, namespace, pvcName string) {
	assetsDir := filepath.Join(repoRoot, "examples", "openshift", "tests", "assets")
	entries, err := os.ReadDir(assetsDir)
	Expect(err).NotTo(HaveOccurred())
	Expect(entries).NotTo(BeEmpty(), "No test assets found in %s", assetsDir)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helperPodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "seeder",
				Image:   helperImage,
				Command: []string{"sleep", "3600"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "data",
					MountPath: "/data",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			}},
		},
	}
	Expect(k8sClient.Create(ctx, pod)).To(Succeed())

	Eventually(func(g Gomega) {
		p := &corev1.Pod{}
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: helperPodName, Namespace: namespace}, p)).To(Succeed())
		g.Expect(p.Status.Phase).To(Equal(corev1.PodRunning))
	}).WithTimeout(2 * time.Minute).WithPolling(PollInterval).Should(Succeed())

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pdf") {
			continue
		}
		src := filepath.Join(assetsDir, entry.Name())
		dst := fmt.Sprintf("%s/%s:/data/%s", namespace, helperPodName, entry.Name())
		cmd := exec.Command("kubectl", "cp", src, dst, "-c", "seeder")
		output, cpErr := cmd.CombinedOutput()
		Expect(cpErr).NotTo(HaveOccurred(), "kubectl cp failed for %s: %s", entry.Name(), string(output))
	}

	verifyCmd := exec.Command("kubectl", "exec", "-n", namespace, helperPodName, "-c", "seeder", "--", "ls", "-la", "/data/")
	output, err := verifyCmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("Uploaded test assets:\n%s\n", string(output))
}
