package sparkoperatormodule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	platformv1alpha1 "github.com/opendatahub-io/spark-operator-module/pkg/apis/v1alpha1"
)

type controllerArgumentOverride struct {
	prefix string
	value  string
}

// applyControllerConfiguration writes explicitly configured controller
// settings into the rendered Deployment arguments. Fields that are omitted
// retain their manifest defaults.
func applyControllerConfiguration(resources []unstructured.Unstructured,
	configuration *platformv1alpha1.ControllerConfiguration) error {
	if configuration == nil {
		return nil
	}

	overrides, err := controllerArgumentOverrides(configuration)
	if err != nil {
		return err
	}
	if len(overrides) == 0 {
		return nil
	}

	for i := range resources {
		if resources[i].GetKind() != deploymentKind || resources[i].GetName() != sparkOperatorControllerDeployment {
			continue
		}
		containers, found, err := unstructured.NestedSlice(resources[i].Object, "spec", "template", "spec", "containers")
		if err != nil {
			return fmt.Errorf("reading containers on Deployment/%s: %w", resources[i].GetName(), err)
		}
		if !found || len(containers) == 0 {
			return fmt.Errorf("Deployment/%s has no containers", resources[i].GetName())
		}

		for ci, raw := range containers {
			container, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("Deployment/%s containers[%d] is not an object", resources[i].GetName(), ci)
			}
			if container["name"] != controllerContainerName {
				continue
			}
			args, err := controllerArguments(container, resources[i].GetName(), ci)
			if err != nil {
				return err
			}
			for _, override := range overrides {
				args = setControllerArgument(args, override.prefix, override.value)
			}
			container["args"] = args
			containers[ci] = container
			if err := unstructured.SetNestedSlice(resources[i].Object, containers, "spec", "template", "spec", "containers"); err != nil {
				return fmt.Errorf("writing containers on Deployment/%s: %w", resources[i].GetName(), err)
			}
			return nil
		}
		return fmt.Errorf("Deployment/%s has no %q container", resources[i].GetName(), controllerContainerName)
	}

	return fmt.Errorf("Deployment/%s was not rendered", sparkOperatorControllerDeployment)
}

func controllerArgumentOverrides(configuration *platformv1alpha1.ControllerConfiguration) ([]controllerArgumentOverride, error) {
	overrides := make([]controllerArgumentOverride, 0, 6)
	if configuration.Workers != nil {
		overrides = append(overrides, controllerArgumentOverride{"--controller-threads=", strconv.FormatInt(int64(*configuration.Workers), 10)})
	}
	if configuration.WorkqueueRateLimiter != nil {
		workqueue := configuration.WorkqueueRateLimiter
		if workqueue.BucketQPS != nil {
			overrides = append(overrides, controllerArgumentOverride{"--workqueue-ratelimiter-bucket-qps=", strconv.FormatInt(int64(*workqueue.BucketQPS), 10)})
		}
		if workqueue.BucketSize != nil {
			overrides = append(overrides, controllerArgumentOverride{"--workqueue-ratelimiter-bucket-size=", strconv.FormatInt(int64(*workqueue.BucketSize), 10)})
		}
		if workqueue.MaxDelay != nil {
			duration, err := time.ParseDuration(*workqueue.MaxDelay)
			if err != nil || duration <= 0 {
				return nil, fmt.Errorf("controller workqueue maxDelay must be greater than zero")
			}
			overrides = append(overrides, controllerArgumentOverride{"--workqueue-ratelimiter-max-delay=", *workqueue.MaxDelay})
		}
	}
	if configuration.KubeAPIQPS != nil {
		overrides = append(overrides, controllerArgumentOverride{"--kube-api-qps=", strconv.FormatInt(int64(*configuration.KubeAPIQPS), 10)})
	}
	if configuration.KubeAPIBurst != nil {
		overrides = append(overrides, controllerArgumentOverride{"--kube-api-burst=", strconv.FormatInt(int64(*configuration.KubeAPIBurst), 10)})
	}
	return overrides, nil
}

func controllerArguments(container map[string]any, deploymentName string, containerIndex int) ([]any, error) {
	rawArgs, ok := container["args"]
	if !ok {
		return []any{}, nil
	}
	args, ok := rawArgs.([]any)
	if !ok {
		return nil, fmt.Errorf("Deployment/%s containers[%d].args is not a list", deploymentName, containerIndex)
	}
	for ai, raw := range args {
		if _, ok := raw.(string); !ok {
			return nil, fmt.Errorf("Deployment/%s containers[%d].args[%d] is not a string", deploymentName, containerIndex, ai)
		}
	}
	return args, nil
}

func setControllerArgument(args []any, prefix, value string) []any {
	updated := make([]any, 0, len(args)+1)
	replaced := false
	for _, raw := range args {
		arg := raw.(string)
		if !strings.HasPrefix(arg, prefix) {
			updated = append(updated, arg)
			continue
		}
		if !replaced {
			updated = append(updated, prefix+value)
			replaced = true
		}
	}
	if !replaced {
		updated = append(updated, prefix+value)
	}
	return updated
}
