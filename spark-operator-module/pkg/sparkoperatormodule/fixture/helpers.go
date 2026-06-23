package fixture

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/opendatahub-io/spark-operator-module/pkg/apis/v1alpha1"
)

func ReadyDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main", Image: "busybox:latest"},
					},
				},
			},
		},
	}
}

func CreateReadyDeployment(ctx context.Context, cli client.Client, name, namespace string) {
	dep := ReadyDeployment(name, namespace)
	_ = client.IgnoreAlreadyExists(cli.Create(ctx, dep))
	dep.Status.AvailableReplicas = 1
	dep.Status.Replicas = 1
	dep.Status.ReadyReplicas = 1
	_ = cli.Status().Update(ctx, dep)
}

func TriggerReconcile(ctx context.Context, cli client.Client, cr *platformv1alpha1.SparkOperator, trigger string) {
	_ = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := cli.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
			return err
		}
		if cr.Annotations == nil {
			cr.Annotations = map[string]string{}
		}
		cr.Annotations["test/trigger"] = trigger
		return cli.Update(ctx, cr)
	})
}
