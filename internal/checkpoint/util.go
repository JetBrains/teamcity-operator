package checkpoint

import (
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
)

func getInitialStageFromInstance(teamcity TeamCity) Stage {
	if teamcity.IsMultiNode() {
		return ReplicaReady
	}
	return UpdateInitiated
}
