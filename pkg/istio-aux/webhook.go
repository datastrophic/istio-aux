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

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,admissionReviewVersions=v1,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,sideEffects=None,name=istio-aux.datastrophic.io
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
package istioaux

import (
	"context"
	"encoding/json"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PodMutator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := ctrl.Log.WithName("webhook")
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	/* not checking for the target namespace existense and required labels
	as they can be configured via a namespaceSelector in the MutatingWebhookConfiguration:
	  namespaceSelector:
		matchExpressions:
		- key: io.datastrophic/istio-aux
	      operator: In
		  values: ["enabled"]
		- key: istio-injection
	      operator: In
		  values: ["enabled"]
	*/

	logger.WithName("webhook").Info("processing", "pod-generate-name", pod.GenerateName, "pod-name", pod.ObjectMeta.Name)
	SetMetadata(&pod.ObjectMeta)
	logger.WithName("webhook").Info("processed", "pod-generate-name", pod.GenerateName, "pod-name", pod.ObjectMeta.Name)

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
