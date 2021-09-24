// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,admissionReviewVersions=v1,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,sideEffects=None,name=istio-aux.datastrophic.io
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;patch;delete
package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PodMutator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	// TODO: check namespace labels and if the parent is a job (based on the app flag)
	logger.WithName("webhook").Info(fmt.Sprintf("Processing pod %s", pod.ObjectMeta.Name))
	SetMetadata(&pod.ObjectMeta)
	InjectAuxContainer(pod)

	marshaledPod, err := json.Marshal(pod)
	logger.WithName("webhook").Info(fmt.Sprintf("Pod processed:\n%s", string(marshaledPod)))
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (a *PodMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
