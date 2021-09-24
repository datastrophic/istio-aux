package pkg

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	logger = ctrl.Log.WithName("istio-aux")
)

func InjectAuxContainer(pod *corev1.Pod) {
	sidecar := getAuxContainer(DefaultAuxContainerName, DefaultAuxContainerImage, DefaultAuxContainerPollIntervalSeconds)
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)
}

func getAuxContainer(containerName string, containerImage string, pollIntervalSeconds int32) corev1.Container {
	pollingCommand := fmt.Sprintf("while $(curl --output /dev/null --silent --head --fail http://localhost:15021/healthz/ready); do sleep %d; done;", pollIntervalSeconds)

	return corev1.Container{
		Name:    containerName,
		Image:   containerImage,
		Command: []string{"/bin/sh", "-c"},
		Args:    []string{pollingCommand},
	}
}

func SetMetadata(meta *metav1.ObjectMeta) {
	setLabel(meta, IstioAuxLabelName, IstioAuxLabelValue)
	setAnnotation(meta, IstioPodAnnotationName, IstioPodAnnotationValue)
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

func setAnnotation(meta *metav1.ObjectMeta, key string, value string) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}

	if val, found := meta.Labels[key]; found {
		logger.Info(fmt.Sprintf("pod %s already has annotation %s set to %s", meta.Name, key, val))
	} else {
		meta.Annotations[key] = value
	}
}

func CheckNamespaceMeta(meta *metav1.ObjectMeta) error {
	if meta.Labels == nil {
		return fmt.Errorf("required labels are not provided for namespace %s", meta.Name)
	}

	err := checkNamespaceLabel(meta, IstioAuxLabelName, IstioAuxLabelValue)
	if err != nil {
		return err
	}

	err = checkNamespaceLabel(meta, IstioInjectionLabelName, IstioInjectionLabelValue)
	if err != nil {
		return err
	}

	return nil
}

func checkNamespaceLabel(meta *metav1.ObjectMeta, key string, value string) error {
	val, ok := meta.Labels[key]
	if !ok {
		return fmt.Errorf("required label %s is not provided for namespace %s", key, meta.Name)
	}
	if value != val {
		return fmt.Errorf("required label %s is provided but set to %s for namespace %s. expected value: %s", key, val, meta.Name, value)
	}
	return nil
}
