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

package controllers

import (
	"context"
	"fmt"
	"os"

	istio_aux "com.github/datastrophic/istio-aux/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/exec,verbs=get;create;
//+kubebuilder:rbac:groups=core,resources=pods/log,verbs=get;list
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}

	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch Pod", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statuses := pod.Status.ContainerStatuses
	if len(statuses) >= 2 && statuses[0].Name == "istio-proxy" {
		istioState := statuses[0].State
		payloadState := statuses[1].State

		// This needs additional triaging to verify the behavior when the main container restart/failures happen
		if istioState.Running != nil && payloadState.Terminated != nil && payloadState.Terminated.ExitCode == 0 {
			logger.Info("the payload container is terminated, shutting down istio proxy", "pod", req.NamespacedName.String())
			execReq := r.RESTClient.
				Post().
				Namespace(pod.Namespace).
				Resource("pods").
				Name(pod.Name).
				SubResource("exec").
				VersionedParams(&corev1.PodExecOptions{
					Container: "istio-proxy",
					Command:   []string{"sh", "-c", "curl -sf -XPOST http://127.0.0.1:15020/quitquitquit"},
					Stdin:     true,
					Stdout:    true,
					Stderr:    true,
				}, runtime.NewParameterCodec(r.Scheme))

			exec, err := remotecommand.NewSPDYExecutor(r.RESTConfig, "POST", execReq.URL())
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while creating remote command executor: %v", err)
			}

			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    false,
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while running exec: %v", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(getPredicate()).
		Complete(r)
}

func getPredicate() predicate.Predicate {
	eventTypesPredicate := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return true },
	}

	labelSelector, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchLabels: map[string]string{istio_aux.IstioAuxLabelName: istio_aux.IstioAuxLabelValue}})
	if err != nil {
		ctrl.Log.Error(err, "unable to create label selector predicate")
		return eventTypesPredicate
	}

	return predicate.And(eventTypesPredicate, labelSelector)
}
