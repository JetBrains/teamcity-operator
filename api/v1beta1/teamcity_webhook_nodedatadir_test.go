package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateNodeDataDirAbsentIsOk(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateNodeDataDirGreenfield(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateNodeDataDirClaimNameRequiresTemplate(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.NodeDataDirClaimName = "existing-claim"
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nodeDataDirVolumeClaim must be set")
}

func TestValidateNodeDataDirBlankClaimName(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	tc.Spec.MainNode.Spec.NodeDataDirClaimName = "   "
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be blank")
}

func TestValidateNodeDataDirNameCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	tc.Spec.NodeDataDirVolumeClaim.Name = tc.Spec.DataDirVolumeClaim.Name
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides")
}

func TestValidateNodeDataDirDerivedNameCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Name = "main"
	tc.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		{
			Name: "node-data-dir-main",
			VolumeMount: corev1.VolumeMount{
				Name:      "extra",
				MountPath: "/extra",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "derived PVC name")
}

func TestValidateNodeDataDirDuplicateClaimNames(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	tc.Spec.MainNode.Spec.NodeDataDirClaimName = "shared-claim"
	tc.Spec.SecondaryNodes = []Node{
		{
			Name: "secondary",
			Spec: NodeSpec{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("500m"),
					"memory": resource.MustParse("1Gi"),
				},
				NodeDataDirClaimName: "shared-claim",
			},
		},
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same node data PVC")
}

func TestValidateNodeDataDirMountPathCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	tc.Spec.NodeDataDirVolumeClaim.VolumeMount.MountPath = tc.Spec.DataDirVolumeClaim.VolumeMount.MountPath
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumeMount.mountPath")
}

func TestValidateNodeDataDirVolumeNameCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	tc.Spec.NodeDataDirVolumeClaim.VolumeMount.Name = tc.Spec.DataDirVolumeClaim.VolumeMount.Name
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumeMount.name")
}

func TestValidateNodeDataDirStartupPropertyRejected(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.NodeDataDirVolumeClaim = validNodeDataDirTemplate()
	tc.Spec.StartupPropertiesConfig = map[string]string{
		"teamcity.node.data.path": "/wrong",
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "startupPropertiesConfig")
}

func validNodeDataDirTemplate() *CustomPersistentVolumeClaim {
	return &CustomPersistentVolumeClaim{
		Name: "node-data-dir",
		VolumeMount: corev1.VolumeMount{
			Name:      "node-data-dir",
			MountPath: "/mnt/node-data-dir",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}
