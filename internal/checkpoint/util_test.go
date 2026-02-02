package checkpoint

import (
	"testing"

	"git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestGetInitialStageFromInstance(t *testing.T) {
	t.Run("returns ReplicaReady for multi-node setup", func(t *testing.T) {
		instance := v1beta1.TeamCity{
			Spec: v1beta1.TeamCitySpec{
				MainNode: v1beta1.Node{Name: "main"},
				SecondaryNodes: []v1beta1.Node{
					{Name: "secondary-1"},
				},
			},
		}

		stage := getInitialStageFromInstance(instance)

		assert.Equal(t, ReplicaReady, stage)
	})

	t.Run("returns UpdateInitiated for single-node setup", func(t *testing.T) {
		instance := v1beta1.TeamCity{
			Spec: v1beta1.TeamCitySpec{
				MainNode: v1beta1.Node{Name: "main"},
			},
		}

		stage := getInitialStageFromInstance(instance)

		assert.Equal(t, UpdateInitiated, stage)
	})

	t.Run("returns UpdateInitiated when secondary nodes is empty", func(t *testing.T) {
		instance := v1beta1.TeamCity{
			Spec: v1beta1.TeamCitySpec{
				MainNode:       v1beta1.Node{Name: "main"},
				SecondaryNodes: []v1beta1.Node{},
			},
		}

		stage := getInitialStageFromInstance(instance)

		assert.Equal(t, UpdateInitiated, stage)
	})
}
