package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestDefaultFillsMissingMountNamesFromVolumeNames(t *testing.T) {
	tc := &TeamCity{
		Spec: TeamCitySpec{
			Volumes: []TeamCityVolume{
				{Volume: corev1.Volume{Name: "extra-config"}, Mount: &corev1.VolumeMount{MountPath: "/mnt/extra-config"}},
				{Volume: corev1.Volume{Name: "git-key"}},
				{Volume: corev1.Volume{Name: "named"}, Mount: &corev1.VolumeMount{Name: "custom", MountPath: "/mnt/named"}},
			},
			MainNode: Node{Spec: NodeSpec{Volumes: []TeamCityVolume{
				{Volume: corev1.Volume{Name: "main-only"}, Mount: &corev1.VolumeMount{MountPath: "/mnt/main"}},
			}}},
			SecondaryNodes: []Node{{Spec: NodeSpec{Volumes: []TeamCityVolume{
				{Volume: corev1.Volume{Name: "nodeconfig"}, Mount: &corev1.VolumeMount{MountPath: "/mnt/nodeconfig"}},
			}}}},
		},
	}

	tc.Default()

	assert.Equal(t, "extra-config", tc.Spec.Volumes[0].Mount.Name)
	assert.Nil(t, tc.Spec.Volumes[1].Mount)
	assert.Equal(t, "custom", tc.Spec.Volumes[2].Mount.Name)
	assert.Equal(t, "main-only", tc.Spec.MainNode.Spec.Volumes[0].Mount.Name)
	assert.Equal(t, "nodeconfig", tc.Spec.SecondaryNodes[0].Spec.Volumes[0].Mount.Name)
}
