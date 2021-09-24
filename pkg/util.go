package pkg

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	logger = ctrl.Log.WithName("tfjob")
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
	if val, found := meta.Labels[IstioAuxLabelName]; found {
		logger.Info(fmt.Sprintf("Job %s already has label '%s' set to '%s'", meta.Name, IstioAuxLabelName, val))
	} else {
		meta.Labels[IstioAuxLabelName] = IstioAuxLabelValue
	}

	if val, found := meta.Annotations[IstioPodAnnotationName]; found {
		logger.Info(fmt.Sprintf("Job %s already has label '%s' set to '%s'", meta.Name, IstioPodAnnotationName, val))
	} else {
		meta.Labels[IstioPodAnnotationName] = IstioPodAnnotationValue
	}
}
