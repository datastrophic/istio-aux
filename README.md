# Istio AUX Controller
A simple controller that configures Istio Proxy for matching payloads to delay the application start
until the Proxy is ready and ensures the completion of Job-like resources by shutting down the Proxy
upon the main container exit.

## Quick Start
### Prerequisites
You should have a Kubernetes cluster available, kind will suffice but ensure the Docker daemon has sufficient resources to accommodate for cert-manager, Istio, Kubeflow training operator, and run a two-pod TFJob (8CPU, 8GB RAM should be sufficient). The following software is required:
* [kind](https://kind.sigs.k8s.io/)
* [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
* [istioctl](https://istio.io/latest/docs/setup/getting-started/#download)

### Cluster setup
The cluster setup is pretty straightforward and includes installation of all the required dependencies. Will use the Composite Operator that supports all types of training jobs (former TensorFlow Operator) and this step is not required if you plan to use vanilla Kubernetes `Jobs`.
```
kind create cluster

# wait for node(s) to become ready
kubectl wait --for condition=Ready nodes

# install cert-manager
kubectl create -f https://github.com/jetstack/cert-manager/releases/download/v1.5.3/cert-manager.yaml

# wait for pods to become ready
kubectl wait --for=condition=Ready pods --namespace cert-manager

# install istio
istioctl install --set profile=demo -y

# install the training operator
kubectl apply -k "github.com/kubeflow/tf-operator.git/manifests/overlays/standalone?ref=master"

# install the Istio AUX controller

```

### A `TFJob` example
First, let's create a TFJob that will be used for testing, enable Istio injection for the default namespace, and submit the job:
```
kubectl label namespace default istio-injection=enabled

cat <<EOF >./tfjob.yaml
apiVersion: kubeflow.org/v1
kind: TFJob
metadata:
  name: mnist
spec:
  tfReplicaSpecs:
    Worker:
      replicas: 2
      restartPolicy: OnFailure
      template:
        spec:
          containers:
          - name: tensorflow
            image: datastrophic/tensorflow:2.6.0-mnist
            command: ['python', '-u', 'mnist.py']
EOF

kubectl create -f tfjob.yaml

kubectl get pods -w
```
We'll see that the pods will eventually get stuck in `NotReady` state with one container still running.

Now let's enable the Istio AUX Controller for the default namespace and redeploy the `TFJob` one more time.
```
kubectl delete -f tfjob.yaml

kubectl label namespace default io.datastrophic/istio-aux=enabled

kubectl create -f tfjob.yaml

kubectl get pods -w
```
This time, all the pods reached the `Completed` state.

In the meantime, the Istio AUX Controller logs contain an output like this:
```

```

## How it works
Istio AUX contains a MutatingAdmissionWebhook that mutates the pods submitted to namespaces with specific labels and adds an Istio-specific annotation to Pods:
```
proxy.istio.io/config: "holdApplicationUntilProxyStarts: true"
```

That way, Istio Operator will take care of rearranging the sidecars and delaying the first non-Istio container start until the proxy is ready. This can also be solved, by setting the same Istio Proxy property globally, however, it is `false` by default and it's not clear whether this setting can impact other existing deployments outside Kubeflow.

Another part of Istio AUX is a Pod Controller that is also scoped to namespaces with specific labels and subscribed to Pod Update events. All the container status changes trigger the reconciliation, and the controller keeps checking what containers are still running in the Pod. Once there's only one left and it is Istio Proxy, the controller execs into a pod and runs `curl -sf -XPOST http://127.0.0.1:15020/quitquitquit` in it. Istio Proxy container image has `curl` pre-installed so there's no need for an additional binary or a sidecar to terminate the proxy.

The termination heuristic is pretty naive but it is easy to extend it to a more sophisticated version e.g. checking against a list of container names that have to exit prior to terminating the Proxy.

## Development guide
The project is created with Kubebuilder 1.3.1, consult the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) docs for the details.
