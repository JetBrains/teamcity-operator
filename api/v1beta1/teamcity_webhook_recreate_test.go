package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateUpdateServiceNameRequiresAnnotation(t *testing.T) {
	old := validTeamCityForWebhookTest()
	updated := old.DeepCopy()
	updated.Spec.MainNode.Spec.ServiceName = "headless-svc"

	_, err := updated.ValidateUpdate(old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), AllowStsRecreateAnnotationKey)
}

func TestValidateUpdateServiceNameWithAnnotationWarns(t *testing.T) {
	old := validTeamCityForWebhookTest()
	updated := old.DeepCopy()
	updated.Annotations = map[string]string{
		AllowStsRecreateAnnotationKey: AllowStsRecreateAnnotationValue,
	}
	updated.Spec.MainNode.Spec.ServiceName = "headless-svc"

	warnings, err := updated.ValidateUpdate(old)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "will be recreated")
}

func validTeamCityForWebhookTest() *TeamCity {
	return &TeamCity{
		ObjectMeta: metav1.ObjectMeta{Name: "teamcity"},
		Spec: TeamCitySpec{
			XmxPercentage: 95,
			MainNode: Node{
				Name: "main",
				Spec: NodeSpec{
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("500m"),
						"memory": resource.MustParse("1Gi"),
					},
				},
			},
			DataDirVolumeClaim: CustomPersistentVolumeClaim{
				Name: "data",
				VolumeMount: corev1.VolumeMount{
					Name:      "data",
					MountPath: "/storage",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
}
