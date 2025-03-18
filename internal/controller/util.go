package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/checkpoint"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
)

func getTeamCityObjectE(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (teamcity TeamCity, err error) {
	if err := r.Get(ctx, namespacedName, &teamcity); err != nil {
		return teamcity, err
	}
	return teamcity, nil
}

func updateTeamCityObjectStatusE(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName, state string, status string) (err error) {
	var teamcity TeamCity
	if teamcity, err = getTeamCityObjectE(r, ctx, namespacedName); err != nil {
		return err
	}
	teamcityStatus := TeamCityStatus{State: state, Message: status}
	if !reflect.DeepEqual(teamcity.Status, teamcityStatus) {
		teamcity.Status = teamcityStatus
		err = r.Status().Update(context.Background(), &teamcity)
		if err != nil {
			return err
		}
	}
	return nil
}

func getStatefulSetByName(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (statefulSet v1.StatefulSet, error error) {
	if err := r.Get(ctx, namespacedName, &statefulSet); err != nil {
		return statefulSet, err
	}
	return statefulSet, nil
}

func isNewestGeneration(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (bool bool, err error) {
	var statefulSet v1.StatefulSet
	if statefulSet, err = getStatefulSetByName(r, ctx, namespacedName); err != nil {
		return false, err
	}
	if statefulSet.Generation != statefulSet.Status.ObservedGeneration {
		return false, nil
	}
	return true, nil
}

func isNodeUpdateFinished(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (bool bool, err error) {
	var statefulSet v1.StatefulSet
	if statefulSet, err = getStatefulSetByName(r, ctx, namespacedName); err != nil {
		return false, err
	}
	if statefulSet.Status.CurrentRevision == statefulSet.Status.UpdateRevision && statefulSet.Status.ReadyReplicas == int32(1) {
		return true, nil
	}
	return false, nil
}

func doesNodesUpdateChangeStatefulSetSpec(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	for _, node := range instance.GetAllNodes() {
		var nodeStatefulSet v1.StatefulSet
		if err := r.Get(ctx, node.GetNamespacedNameFromNamespace(instance.Namespace), &nodeStatefulSet); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if resource.ChangesRequireNodeStatefulSetRestart(instance, node, &nodeStatefulSet) {
			return true, nil
		}

	}
	return false, nil
}

func ongoingZeroDowntimeUpgrade(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) bool {
	initialCheckpoint := checkpoint.NewCheckpoint(r.Client, *instance)
	_, err := initialCheckpoint.FetchCurrentStageFromCluster(ctx)
	if err != nil {
		return false
	}
	return true
}
