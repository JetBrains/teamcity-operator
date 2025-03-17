package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/checkpoint"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func HandleStageChange(r *TeamcityReconciler, ctx context.Context, instance *TeamCity, currentStage checkpoint.Stage) (bool, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Current update stage is " + currentStage.String())
	switch currentStage {
	case checkpoint.UpdateInitiated:
		result, err := HandleUpdateInitiated(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	case checkpoint.ReplicaCreated:
		result, err := HandleReplicaCreated(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	case checkpoint.ReplicaStarting:
		result, err := HandleReplicaStarting(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	case checkpoint.ReplicaReady:
		result, err := HandleReplicaReady(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	case checkpoint.MainShuttingDown:
		result, err := HandleMainShuttingDown(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	case checkpoint.MainReady:
		result, err := HandleMainReady(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	case checkpoint.UpdateFinished:
		result, err := HandleUpdateFinished(r, ctx, instance)
		if err != nil {
			return false, err
		}
		return result, nil
	}
	return false, nil
}

func HandleUpdateInitiated(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	err := DoCheckpointE(r, ctx, instance, checkpoint.ReplicaCreated)
	if err != nil {
		return false, err
	}
	return true, nil
}
func HandleReplicaCreated(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	var mainStatefulSet v1.StatefulSet
	if err := r.Get(ctx, instance.Spec.MainNode.GetNamespacedNameFromNamespace(instance.Namespace), &mainStatefulSet); err != nil {
		return false, err
	}
	roStatefulSet := resource.BuildROStatefulSet(instance)
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var apiError error
		_, apiError = controllerutil.CreateOrUpdate(ctx, r.Client, roStatefulSet, func() error {
			return resource.UpdateROStatefulSet(r.Scheme, instance, &mainStatefulSet, roStatefulSet)
		})
		return apiError
	})

	err = DoCheckpointE(r, ctx, instance, checkpoint.ReplicaStarting)
	if err != nil {
		return false, err
	}
	return true, nil
}
func HandleReplicaStarting(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	roStatefulSetName := resource.GetROStatefulSetNamespacedName(instance)
	var roStatefulSet v1.StatefulSet
	if err := r.Get(ctx, roStatefulSetName, &roStatefulSet); err != nil {
		return false, err
	}
	if roStatefulSet.Status.AvailableReplicas > 0 {
		err := DoCheckpointE(r, ctx, instance, checkpoint.ReplicaReady)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}
func HandleReplicaReady(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	err := DoCheckpointE(r, ctx, instance, checkpoint.MainShuttingDown)
	if err != nil {
		return false, err
	}
	return false, err
}
func HandleMainShuttingDown(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	mainStatefulSetName := instance.Spec.MainNode.GetNamespacedNameFromNamespace(instance.Namespace)
	var mainStatefulSet v1.StatefulSet
	if err := r.Get(ctx, mainStatefulSetName, &mainStatefulSet); err != nil {
		return false, err
	}

	if mainStatefulSet.Status.AvailableReplicas > 0 && isStatefulSetNewestGeneration(&mainStatefulSet) {
		err := DoCheckpointE(r, ctx, instance, checkpoint.MainReady)
		if err != nil {
			return false, err
		}
	}
	return true, nil

}
func HandleMainReady(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	roStatefulSetName := resource.GetROStatefulSetNamespacedName(instance)
	var roStatefulSet v1.StatefulSet
	roExists := true
	if err := r.Get(ctx, roStatefulSetName, &roStatefulSet); err != nil {
		if !errors.IsNotFound(err) {
			return false, nil
		} else {
			roExists = false
		}
	}
	if roExists {
		if err := r.Delete(ctx, &roStatefulSet); err != nil {
			if !errors.IsNotFound(err) {
				return false, err
			}
		}
	}

	err := DoCheckpointE(r, ctx, instance, checkpoint.UpdateFinished)
	if err != nil {
		return false, err
	}
	return true, nil
}
func HandleUpdateFinished(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (bool, error) {
	err := DeleteCheckPoint(r, ctx, instance)
	if err != nil {
		return false, err
	}
	return false, nil
}
