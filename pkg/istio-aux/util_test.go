package istioaux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetAnnotation(t *testing.T) {

	t.Run("it=sets the default annotation when no annotations on pod", func(tt *testing.T) {
		objectMeta := &metav1.ObjectMeta{}
		SetMetadata(objectMeta)
		assert.NotNil(tt, objectMeta.Annotations)
		assert.Contains(tt, objectMeta.Annotations, IstioPodAnnotationName)
		assert.Equal(tt, "holdApplicationUntilProxyStarts: true\n", toYaml(tt, IstioPodAnnotationValue))
	})

	t.Run("it=merges with existing annotation", func(tt *testing.T) {
		existingAnnotation := map[string]interface{}{
			"proxyMetadata": map[string]interface{}{
				"OUTPUT_CERTS": "/etc/istio-output-certs",
			},
		}
		expectedAnnotation := map[string]interface{}{
			"holdApplicationUntilProxyStarts": true,
			"proxyMetadata": map[string]interface{}{
				"OUTPUT_CERTS": "/etc/istio-output-certs",
			},
		}
		objectMeta := &metav1.ObjectMeta{
			Annotations: map[string]string{
				IstioPodAnnotationName: toYaml(tt, existingAnnotation),
			},
		}
		SetMetadata(objectMeta)
		assert.NotNil(tt, objectMeta.Annotations)
		assert.Contains(tt, objectMeta.Annotations, IstioPodAnnotationName)
		assert.Equal(tt, "holdApplicationUntilProxyStarts: true\nproxyMetadata:\n  OUTPUT_CERTS: /etc/istio-output-certs\n", toYaml(tt, expectedAnnotation))
	})

	t.Run("it=does not override when pods annotation contained a value", func(tt *testing.T) {
		existingAnnotation := map[string]interface{}{
			"holdApplicationUntilProxyStarts": false,
			"proxyMetadata": map[string]interface{}{
				"OUTPUT_CERTS": "/etc/istio-output-certs",
			},
		}
		objectMeta := &metav1.ObjectMeta{
			Annotations: map[string]string{
				IstioPodAnnotationName: toYaml(tt, existingAnnotation),
			},
		}
		SetMetadata(objectMeta)
		assert.NotNil(tt, objectMeta.Annotations)
		assert.Contains(tt, objectMeta.Annotations, IstioPodAnnotationName)
		assert.Equal(tt, "holdApplicationUntilProxyStarts: false\nproxyMetadata:\n  OUTPUT_CERTS: /etc/istio-output-certs\n", toYaml(tt, existingAnnotation))
	})

}

func toYaml(t *testing.T, data map[string]interface{}) string {
	bs, err := yaml.Marshal(&data)
	assert.Nil(t, err)
	return string(bs)
}
