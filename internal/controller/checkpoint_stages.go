package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/checkpoint"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	ctrl "sigs.k8s.io/controller-runtime"
)

func HandleUpdateStarted(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	//create RO node and apply it
	resource.BuildROStatefulSet(instance)

	err := DoCheckpointE(r, ctx, instance, checkpoint.ReplicaStarting)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
