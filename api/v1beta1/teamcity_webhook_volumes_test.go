package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateVolumesAbsentIsOk(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateVolumesMountedAndUnmounted(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", "/mnt/extra-config"),
		{
			Volume: corev1.Volume{
				Name: "git-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: "git-key"},
				},
			},
		},
	}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateVolumesRejectsEmptyName(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	volume := configMapVolumeForWebhookTest("", "/mnt/extra-config")
	tc.Spec.Volumes = []TeamCityVolume{volume}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Volume name is not set")
}

func TestValidateVolumesRejectsDuplicateNames(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", "/mnt/one"),
		configMapVolumeForWebhookTest("extra-config", "/mnt/two"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with")
}

func TestValidateVolumesAcceptsNodeLevelVolume(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", "/mnt/extra-config"),
	}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateVolumesRejectsNodeLevelEmptyName(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("", "/mnt/extra-config"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Volume name is not set")
}

func TestValidateVolumesRejectsNodeLevelShadowingSpecLevel(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", "/mnt/extra-config"),
	}
	tc.Spec.MainNode.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", "/mnt/other"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with")
}

func TestValidateVolumesRejectsNodeLevelMountPathClashWithSpecLevel(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", "/mnt/shared"),
	}
	tc.Spec.MainNode.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("node-config", "/mnt/shared"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mount.mountPath")
}

func TestValidateVolumesRejectsMountPathCollisionWithDataDir(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("extra-config", tc.Spec.DataDirVolumeClaim.VolumeMount.MountPath),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mount.mountPath")
}

func TestValidateVolumesRejectsVolumeNameCollisionWithDataDir(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest(tc.Spec.DataDirVolumeClaim.VolumeMount.Name, "/mnt/extra-config"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with")
}

func TestValidateVolumesAllowsSameMountPathOnDifferentNodes(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.SecondaryNodes = []Node{secondaryNodeForWebhookTest("secondary")}

	tc.Spec.MainNode.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("config-main", "/mnt/nodeconfig"),
	}
	tc.Spec.SecondaryNodes[0].Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("config-secondary", "/mnt/nodeconfig"),
	}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateVolumesAllowsSameVolumeNameOnDifferentNodes(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.SecondaryNodes = []Node{secondaryNodeForWebhookTest("secondary")}

	tc.Spec.MainNode.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("nodeconfig", "/mnt/nodeconfig"),
	}
	tc.Spec.SecondaryNodes[0].Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("nodeconfig", "/mnt/nodeconfig"),
	}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateVolumesWarnsAboutHostPath(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.Volumes = []TeamCityVolume{
		{
			Volume: corev1.Volume{
				Name: "host-scratch",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/tmp/scratch"},
				},
			},
			Mount: &corev1.VolumeMount{MountPath: "/mnt/scratch"},
		},
	}
	warnings, err := tc.ValidateCreate()
	require.NoError(t, err)
	require.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "hostPath")
}

func TestValidateNodePersistentVolumeClaimsGreenfield(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.SecondaryNodes = []Node{secondaryNodeForWebhookTest("secondary")}
	tc.Spec.MainNode.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest("git-cache-main-node", "git-cache", "/mnt/git-cache"),
	}
	tc.Spec.SecondaryNodes[0].Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest("git-cache-secondary", "git-cache", "/mnt/git-cache"),
	}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateNodePersistentVolumeClaimsRejectsSharedClaimName(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.SecondaryNodes = []Node{secondaryNodeForWebhookTest("secondary")}
	tc.Spec.MainNode.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest("git-cache", "git-cache", "/mnt/git-cache"),
	}
	tc.Spec.SecondaryNodes[0].Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest("git-cache", "git-cache", "/mnt/git-cache"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve to the same PVC")
}

func TestValidateNodePersistentVolumeClaimsRejectsCollisionWithDataDir(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest(tc.Spec.DataDirVolumeClaim.Name, "git-cache", "/mnt/git-cache"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with")
}

func TestValidateNodePersistentVolumeClaimsRejectsCollisionWithNodeDataDir(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.NodeDataDirVolumeClaim = validNodeDataDirClaim("node-data-dir-main-node")
	tc.Spec.MainNode.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest("git-cache-main-node", "node-data-dir", "/mnt/git-cache"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumeMount.name")
}

func TestValidateNodePersistentVolumeClaimsExistingClaimWithoutSpec(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	claim := nodeClaimForWebhookTest("existing-git-cache", "git-cache", "/mnt/git-cache")
	claim.Spec = corev1.PersistentVolumeClaimSpec{}
	claim.ExistingClaim = true
	tc.Spec.MainNode.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{*claim}
	_, err := tc.ValidateCreate()
	require.NoError(t, err)
}

func TestValidateVolumesRejectsCollisionWithNodeClaim(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{
		*nodeClaimForWebhookTest("git-cache-main-node", "git-cache", "/mnt/git-cache"),
	}
	tc.Spec.Volumes = []TeamCityVolume{
		configMapVolumeForWebhookTest("git-cache", "/mnt/elsewhere"),
	}
	_, err := tc.ValidateCreate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with")
}

func configMapVolumeForWebhookTest(name string, mountPath string) TeamCityVolume {
	return TeamCityVolume{
		Volume: corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "tc-extra-config"},
				},
			},
		},
		Mount: &corev1.VolumeMount{MountPath: mountPath},
	}
}

func nodeClaimForWebhookTest(name string, volumeName string, mountPath string) *CustomPersistentVolumeClaim {
	return &CustomPersistentVolumeClaim{
		Name: name,
		VolumeMount: corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
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

func secondaryNodeForWebhookTest(name string) Node {
	return Node{
		Name: name,
		Spec: NodeSpec{
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("500m"),
				"memory": resource.MustParse("1Gi"),
			},
		},
	}
}
