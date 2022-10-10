/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package istioaux

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"gopkg.in/yaml.v2"
)

var (
	logger = ctrl.Log.WithName("istio-aux")
)

func usingMapCopy(input map[string]interface{}) map[string]interface{} {
	copy := map[string]interface{}{}
	for k, v := range input {
		copy[k] = v
	}
	return copy
}

func SetMetadata(meta *metav1.ObjectMeta) {
	setLabel(meta, IstioAuxLabelName, IstioAuxLabelValue)
	// IstioPodAnnotationValue is no longer const and there's only one copy in memory!
	// So, it's a template do NOT modify it, use it as a template!
	setAnnotation(meta, IstioPodAnnotationName, usingMapCopy(IstioPodAnnotationValue))
}

func setLabel(meta *metav1.ObjectMeta, key string, value string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}

	if val, found := meta.Labels[IstioAuxLabelName]; found {
		logger.Info(fmt.Sprintf("Pod %s already has label %s set to %s", meta.Name, key, val))
	} else {
		meta.Labels[key] = value
	}
}

func setAnnotation(meta *metav1.ObjectMeta, key string, localValue map[string]interface{}) {

	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}

	if val, found := meta.Annotations[key]; found {
		logger.Info(fmt.Sprintf("pod %s already has annotation %s set to %s, merging...", meta.Name, key, val))

		// so, the value is supposed to be yaml...
		existing := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(val), &existing); err == nil {
			for k, v := range existing {
				localValue[k] = v
			}
		} else {
			logger.Error(err, "pod %s error unmarshaling existing %s annotation value %s", meta.Name, key, val)
		}

	}

	if marshaled, err := yaml.Marshal(localValue); err == nil {
		meta.Annotations[key] = string(marshaled)
	} else {
		logger.Error(err, "pod %s error marshaling %s annotation value %v", meta.Name, key, localValue)
	}

}

func NewRESTRequest(client rest.Interface, codec runtime.ParameterCodec, namespace string, pod string) *rest.Request {
	return client.
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "istio-proxy",
			Command:   []string{"sh", "-c", "curl -sf -XPOST http://127.0.0.1:15020/quitquitquit"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
		}, codec)
}

func GetPredicate() predicate.Predicate {
	eventTypesPredicate := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return true },
	}

	labelSelector, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchLabels: map[string]string{IstioAuxLabelName: IstioAuxLabelValue}})
	if err != nil {
		ctrl.Log.Error(err, "unable to create label selector predicate")
		return eventTypesPredicate
	}

	return predicate.And(eventTypesPredicate, labelSelector)
}

func CheckReadyForCleanup(pod *corev1.Pod) bool {
	statuses := pod.Status.ContainerStatuses
	if len(statuses) >= 2 {
		logger.Info("found a pod with istio proxy, checking container statuses", "pod", pod.Name)

		running := make([]string, 0)
		var istioState *corev1.ContainerState = nil

		for _, container := range statuses {
			state := container.State
			if state.Terminated == nil && container.Name != "istio-proxy" {
				running = append(running, container.Name)
				continue
			}

			if container.Name == "istio-proxy" && state.Terminated != nil {
				logger.Info("istio-proxy is already in a terminated state", "pod", pod.Name)
				return false
			} else {
				istioState = &state
			}
		}

		if istioState == nil {
			logger.Info("istio-proxy is not found", "pod", pod.Name)
			return false
		}

		if len(running) > 0 {
			logger.Info("some containers are still running, skipping istio proxy shutdown", "pod", pod.Name, "containers", running)
			return false
		} else {
			logger.Info("the payload containers are terminated, proceeding with the proxy shutdown", "pod", pod.Name)
			return true
		}
	}
	return false
}
