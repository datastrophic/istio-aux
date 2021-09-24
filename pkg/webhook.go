// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,admissionReviewVersions=v1,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,sideEffects=None,name=istio-aux.datastrophic.io
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
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

	// checking the target namespace exists and has required labels - TODO: this can be achieved via NS selector
	namespaces := &corev1.NamespaceList{}
	err = a.Client.List(context.Background(), namespaces)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	var namespace *corev1.Namespace = nil
	for _, ns := range namespaces.Items {
		if ns.ObjectMeta.Name == pod.Namespace {
			namespace = &ns
			break
		}
	}

	if namespace != nil {
		err = CheckNamespaceMeta(&namespace.ObjectMeta)
		if err != nil {
			msg := fmt.Sprintf("target namespace %s doesn't specify required labels. istio-aux will skip mutation. cause: %s", namespace.ObjectMeta.Name, err)
			logger.WithName("webhook").Info(msg)
			return admission.Allowed(msg)
		}
	} else {
		msg := fmt.Sprintf("target namespace %s doesn't exist. istio-aux will skip mutation", namespace.ObjectMeta.Name)
		logger.WithName("webhook").Info(msg)
		return admission.Allowed(msg)
	}

	logger.WithName("webhook").Info(fmt.Sprintf("processing pod %s/%s", namespace.ObjectMeta.Name, pod.ObjectMeta.Name))
	SetMetadata(&pod.ObjectMeta)
	// InjectAuxContainer(pod)  // TODO: revisit after the controller is done
	logger.WithName("webhook").Info(fmt.Sprintf("pod %s/%s processed", namespace.ObjectMeta.Name, pod.ObjectMeta.Name))

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (a *PodMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
