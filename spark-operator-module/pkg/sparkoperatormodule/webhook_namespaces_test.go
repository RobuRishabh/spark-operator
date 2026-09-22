package sparkoperatormodule

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestApplyWebhookJobNamespaces(t *testing.T) {
	g := NewWithT(t)

	resources := []unstructured.Unstructured{
		{
			Object: map[string]any{
				"apiVersion": "admissionregistration.k8s.io/v1",
				"kind":       "MutatingWebhookConfiguration",
				"metadata":   map[string]any{"name": "mutating-webhook-configuration"},
				"webhooks": []any{
					map[string]any{
						"name": "mutate-sparkapplication.sparkoperator.k8s.io",
						"namespaceSelector": map[string]any{
							"matchExpressions": []any{
								map[string]any{
									"key":      "kubernetes.io/metadata.name",
									"operator": "In",
									"values":   []any{"default"},
								},
							},
						},
					},
					map[string]any{
						"name": "mutate-pod.sparkoperator.k8s.io",
					},
				},
			},
		},
		{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata":   map[string]any{"name": "unrelated"},
			},
		},
		{
			Object: map[string]any{
				"apiVersion": "admissionregistration.k8s.io/v1",
				"kind":       "ValidatingWebhookConfiguration",
				"metadata":   map[string]any{"name": "validating-webhook-configuration"},
				"webhooks": []any{
					map[string]any{
						"name": "validate-sparkapplication.sparkoperator.k8s.io",
					},
				},
			},
		},
	}

	g.Expect(applyWebhookJobNamespaces(resources, []string{"default", "spark-bench-a"})).To(Succeed())

	for _, res := range resources {
		if res.GetKind() != mutatingWebhookKind && res.GetKind() != validatingWebhookKind {
			continue
		}
		webhooks, found, err := unstructured.NestedSlice(res.Object, "webhooks")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(found).To(BeTrue())
		for _, raw := range webhooks {
			webhook := raw.(map[string]any)
			selector := webhook["namespaceSelector"].(map[string]any)
			exprs := selector["matchExpressions"].([]any)
			g.Expect(exprs).To(HaveLen(1))
			expr := exprs[0].(map[string]any)
			g.Expect(expr["key"]).To(Equal(namespaceNameLabel))
			g.Expect(expr["operator"]).To(Equal("In"))
			g.Expect(expr["values"]).To(Equal([]any{"default", "spark-bench-a"}))
		}
	}
}

func TestApplyWebhookJobNamespaces_EmptyRejected(t *testing.T) {
	g := NewWithT(t)
	g.Expect(applyWebhookJobNamespaces(nil, nil)).To(HaveOccurred())
}
