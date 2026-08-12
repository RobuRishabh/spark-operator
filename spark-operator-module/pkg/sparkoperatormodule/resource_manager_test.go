package sparkoperatormodule

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckDeploymentsReady_Missing(t *testing.T) {
	g := NewWithT(t)

	cli := fake.NewClientBuilder().Build()
	err := checkDeploymentsReady(context.Background(), cli, "opendatahub", []string{"spark-operator-controller"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("spark-operator-controller"))
}

func TestCheckDeploymentsReady_Available(t *testing.T) {
	g := NewWithT(t)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "spark-operator-controller",
			Namespace: "opendatahub",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "spark-operator-controller"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "spark-operator-controller"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "test"}}},
			},
		},
	}
	dep.Status.AvailableReplicas = 1
	dep.Status.UpdatedReplicas = 1

	cli := fake.NewClientBuilder().WithObjects(dep).WithStatusSubresource(dep).Build()
	err := checkDeploymentsReady(context.Background(), cli, "opendatahub", []string{"spark-operator-controller"})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestCheckDeploymentsReady_ExistsButUnavailable(t *testing.T) {
	g := NewWithT(t)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "spark-operator-controller",
			Namespace:  "opendatahub",
			Generation: 1,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "spark-operator-controller"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "spark-operator-controller"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "test"}}},
			},
		},
	}
	dep.Status.AvailableReplicas = 0
	dep.Status.UpdatedReplicas = 0
	dep.Status.ObservedGeneration = 1

	cli := fake.NewClientBuilder().WithObjects(dep).WithStatusSubresource(dep).Build()
	err := checkDeploymentsReady(context.Background(), cli, "opendatahub", []string{"spark-operator-controller"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("spark-operator-controller"))
}

func TestCountReadyDeployments_NoneReady(t *testing.T) {
	g := NewWithT(t)

	cli := fake.NewClientBuilder().Build()
	ready, total := countReadyDeployments(context.Background(), cli, "opendatahub")
	g.Expect(total).To(Equal(2))
	g.Expect(ready).To(Equal(0))
}

func TestCountReadyDeployments_PartiallyReady(t *testing.T) {
	g := NewWithT(t)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sparkOperatorControllerDeployment,
			Namespace: "opendatahub",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "ctrl"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "ctrl"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "test"}}},
			},
		},
	}
	dep.Status.AvailableReplicas = 1
	dep.Status.UpdatedReplicas = 1

	cli := fake.NewClientBuilder().WithObjects(dep).WithStatusSubresource(dep).Build()
	ready, total := countReadyDeployments(context.Background(), cli, "opendatahub")
	g.Expect(total).To(Equal(2))
	g.Expect(ready).To(Equal(1))
}

func TestCountReadyDeployments_AllReady(t *testing.T) {
	g := NewWithT(t)

	makeDep := func(name string) *appsv1.Deployment {
		d := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "opendatahub"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "test"}}},
				},
			},
		}
		d.Status.AvailableReplicas = 1
		d.Status.UpdatedReplicas = 1
		return d
	}

	ctrl := makeDep(sparkOperatorControllerDeployment)
	whk := makeDep(sparkOperatorWebhookDeployment)

	cli := fake.NewClientBuilder().WithObjects(ctrl, whk).WithStatusSubresource(ctrl, whk).Build()
	ready, total := countReadyDeployments(context.Background(), cli, "opendatahub")
	g.Expect(total).To(Equal(2))
	g.Expect(ready).To(Equal(2))
}
