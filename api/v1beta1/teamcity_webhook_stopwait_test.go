package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateWarnsAboutStopWaitTimeInNodeEnv(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.StopWaitTimeSeconds = 120
	tc.Spec.SecondaryNodes = []Node{secondaryNodeForWebhookTest("secondary")}
	tc.Spec.SecondaryNodes[0].Spec.Env = []corev1.EnvVar{
		{Name: "OTHER", Value: "1"},
		{Name: "TEAMCITY_STOP_WAIT_TIME", Value: "300"},
	}
	warnings, err := tc.ValidateCreate()
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "teamcity.spec.secondaryNodes[0].spec.env[1]")
	assert.Contains(t, warnings[0], "stopWaitTimeSeconds (120)")
}

func TestValidateNoStopWaitTimeInNodeEnvNoWarning(t *testing.T) {
	tc := validTeamCityForWebhookTest()
	tc.Spec.MainNode.Spec.Env = []corev1.EnvVar{{Name: "OTHER", Value: "1"}}
	warnings, err := tc.ValidateCreate()
	require.NoError(t, err)
	assert.Empty(t, warnings)
}
