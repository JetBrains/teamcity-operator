package controller

import (
	"context"
	jetbrainscomv1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
)

func GetSecretE(r *TeamcityReconciler, secretName string, namespace string) (secret v12.Secret, err error) {
	secretNamespacedName := types.NamespacedName{Namespace: namespace, Name: secretName}
	if err := r.Get(context.TODO(), secretNamespacedName, &secret); err != nil {
		return secret, err
	}
	return secret, nil
}

func GetTeamCityObjectE(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName) (teamcity jetbrainscomv1alpha1.TeamCity, err error) {
	if err := r.Get(ctx, namespacedName, &teamcity); err != nil {
		return teamcity, err
	}
	return teamcity, nil
}

func UpdateTeamCityObjectStatusE(r *TeamcityReconciler, ctx context.Context, namespacedName types.NamespacedName, state string, status string) (err error) {
	var teamcity jetbrainscomv1alpha1.TeamCity
	if teamcity, err = GetTeamCityObjectE(r, ctx, namespacedName); err != nil {
		return err
	}
	teamcityStatus := jetbrainscomv1alpha1.TeamCityStatus{State: state, Message: status}
	if !reflect.DeepEqual(teamcity.Status, teamcityStatus) {
		teamcity.Status = teamcityStatus
		err = r.Status().Update(context.Background(), &teamcity)
		if err != nil {
			return err
		}
	}
	return nil
}
