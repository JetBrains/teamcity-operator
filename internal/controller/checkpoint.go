package controller

import (
	"context"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/checkpoint"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DoCheckpointE(r *TeamcityReconciler, ctx context.Context, instance *TeamCity, desiredStage checkpoint.Stage) (err error) {
	desiredCheckpointCM := checkpoint.BuildCheckpoint(instance.Name, instance.Namespace, desiredStage)

	currentStage, err := GetCurrentStageFromInstance(r, ctx, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			err = createCheckpoint(r, ctx, &desiredCheckpointCM)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if desiredStage < currentStage || desiredStage-currentStage > 1 {
		return fmt.Errorf("illegal stage transition: current stage '%s', desired stage '%s', difference must be 0 or 1",
			currentStage, desiredStage)
	}

	if err := updateCheckpoint(r, ctx, &desiredCheckpointCM); err != nil {
		return err
	}

	return nil

}

func GetCurrentStageFromInstance(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (checkpoint.Stage, error) {
	checkpointCMName := checkpoint.ConstructCheckpointName(instance.Name)
	cm := &v1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Namespace: instance.Namespace, Name: checkpointCMName}, cm)
	if err != nil {
		return checkpoint.Unknown, err
	}
	stageStr, ok := cm.Data["stage"]
	if !ok {
		return checkpoint.Unknown, fmt.Errorf("checkpoint ConfigMap is missing 'stage' key")
	}
	stage, err := checkpoint.ParseStage(stageStr)
	if err != nil {
		return checkpoint.Unknown, err
	}

	return stage, err
}

func createCheckpoint(r *TeamcityReconciler, ctx context.Context, checkpointCM *v1.ConfigMap) error {
	if err := r.Create(ctx, checkpointCM); err != nil {
		return err
	}
	return nil
}

func updateCheckpoint(r *TeamcityReconciler, ctx context.Context, checkpointCM *v1.ConfigMap) error {
	if err := r.Update(ctx, checkpointCM); err != nil {
		return nil
	}
	return nil
}

func DeleteCheckPoint(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) error {
	checkpointCMName := checkpoint.ConstructCheckpointName(instance.Name)
	checkpointCMKey := types.NamespacedName{
		Name:      checkpointCMName,
		Namespace: instance.Namespace,
	}
	checkpointCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      checkpointCMKey.Name,
			Namespace: checkpointCMKey.Namespace,
		},
	}

	if err := r.Get(ctx, checkpointCMKey, checkpointCM); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return nil
	}

	if err := r.Delete(ctx, checkpointCM); err != nil {
		return err
	}

	return nil
}

func OngoingUpdateWithRO(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) bool {
	stage, err := GetCurrentStageFromInstance(r, ctx, instance)
	if err != nil {
		// Handle the case where an error occurs
		return false
	}

	if stage == checkpoint.Unknown {
		// Handle the case where the stage could not be determined
		return false
	}
	return true
}
