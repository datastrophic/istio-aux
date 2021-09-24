package pkg

const IstioAuxLabelName = "io.datastrophic/istio-aux"
const IstioAuxLabelValue = "enabled"

const IstioPodAnnotationName = "proxy.istio.io/config"
const IstioPodAnnotationValue = "holdApplicationUntilProxyStarts: false"

const DefaultAuxContainerName = "istio-aux"
const DefaultAuxContainerImage = "curlimages/curl:7.79.1"
const DefaultAuxContainerPollIntervalSeconds = 1
