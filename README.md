# Istio AUX Controller
A simple controller that configures Istio Proxy for matching payloads to delay the application start
until the Proxy is ready and ensures the completion of Job-like resources by shutting down the Proxy
upon the main container exit.

## Quick Start
TODO:
* [ ] add make targets for a quick cluster-create (optional)
* [ ] refactor manifests into an installable folder so `k create -f folder`
* [ ] the above will require a release target to allow releasing a single YAML
* [ ] install cert manager & Istio
* [ ] deploy example workload
* [ ] label NS and see how everything working now

## Configuration


## How it works
* labels and annotations
* sidecar container with curl to provide the main dependency
* controller doing `exec` - not the most elegant solution but very simple
* link to a blog-post


## Development guide
This is a slightly modified Kubebuilder project with the following differences:
* additional make targets
* kustomize and manifests
* no API as we're working with the core types
