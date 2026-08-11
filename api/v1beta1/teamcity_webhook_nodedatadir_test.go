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
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = validNodeDataDirClaim("node-data-dir-main-node")
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateNodeDataDirExistingClaimWithoutSpec(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = &CustomPersistentVolumeClaim{
		Name:          "existing-node-data-dir",
		ExistingClaim: true,
		VolumeMount: corev1.VolumeMount{
			Name:      "node-data-dir",
			MountPath: "/mnt/node-data-dir",
		},
	}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateNodeDataDirNameCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	claim := validNodeDataDirClaim(tc.Spec.DataDirVolumeClaim.Name)
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = claim
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with")
}

func TestValidateNodeDataDirDuplicateClaimNames(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = validNodeDataDirClaim("shared-claim")
	tc.Spec.SecondaryNodes = []Node{
		{
			Name: "secondary",
			Spec: NodeSpec{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("500m"),
					"memory": resource.MustParse("1Gi"),
				},
				NodeDataDirVolumeClaim: validNodeDataDirClaim("shared-claim"),
			},
		},
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same node data PVC")
}

func TestValidateNodeDataDirMountPathCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	claim := validNodeDataDirClaim("node-data-dir-main-node")
	claim.VolumeMount.MountPath = tc.Spec.DataDirVolumeClaim.VolumeMount.MountPath
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = claim
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumeMount.mountPath")
}

func TestValidateNodeDataDirVolumeNameCollision(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	claim := validNodeDataDirClaim("node-data-dir-main-node")
	claim.VolumeMount.Name = tc.Spec.DataDirVolumeClaim.VolumeMount.Name
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = claim
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumeMount.name")
}

func TestValidateNodeDataDirStartupPropertyRejected(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = validNodeDataDirClaim("node-data-dir-main-node")
	tc.Spec.StartupPropertiesConfig = map[string]string{
		"teamcity.node.data.path": "/wrong",
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "startupPropertiesConfig")
}

func TestValidateNodeDataDirMountMustMatchAcrossNodes(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = validNodeDataDirClaim("node-data-dir-main-node")
	secondaryClaim := validNodeDataDirClaim("node-data-dir-secondary-node")
	secondaryClaim.VolumeMount.MountPath = "/other"
	tc.Spec.SecondaryNodes = []Node{
		{
			Name: "secondary",
			Spec: NodeSpec{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("500m"),
					"memory": resource.MustParse("1Gi"),
				},
				NodeDataDirVolumeClaim: secondaryClaim,
			},
		},
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match other nodes")
}

func validNodeDataDirClaim(name string) *CustomPersistentVolumeClaim {
	return &CustomPersistentVolumeClaim{
		Name: name,
		VolumeMount: corev1.VolumeMount{
			Name:      "node-data-dir",
			MountPath: "/mnt/node-data-dir",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}
