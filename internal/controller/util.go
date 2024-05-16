package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
)

func GetTeamCityObjectE(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (teamcity TeamCity, err error) {
	if err := r.Get(ctx, namespacedName, &teamcity); err != nil {
		return teamcity, err
	}
	return teamcity, nil
}

func UpdateTeamCityObjectStatusE(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName, state string, status string) (err error) {
	var teamcity TeamCity
	if teamcity, err = GetTeamCityObjectE(r, ctx, namespacedName); err != nil {
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

func GetStatefulSetByName(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (statefulSet v1.StatefulSet, error error) {
	if err := r.Get(ctx, namespacedName, &statefulSet); err != nil {
		return statefulSet, err
	}
	return statefulSet, nil
}

func isNewestGeneration(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (bool bool, err error) {
	var statefulSet v1.StatefulSet
	if statefulSet, err = GetStatefulSetByName(r, ctx, namespacedName); err != nil {
		return false, err
	}
	if statefulSet.Generation != statefulSet.Status.ObservedGeneration {
		return false, nil
	}
	return true, nil
}

func isNodeUpdateFinished(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (bool bool, err error) {
	var statefulSet v1.StatefulSet
	if statefulSet, err = GetStatefulSetByName(r, ctx, namespacedName); err != nil {
		return false, err
	}
	updated := statefulSet.Status.CurrentRevision == statefulSet.Status.UpdateRevision
	running := statefulSet.Status.AvailableReplicas == 1
	if updated && running {
		return true, nil
	}
	return false, nil
}
