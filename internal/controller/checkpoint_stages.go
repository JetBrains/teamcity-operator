package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/internal/checkpoint"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func doActionBasedOnCheckpointOrRequeue(r *TeamcityReconciler, ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Current update stage is " + checkpoint.CurrentStage.String())
	switch checkpoint.CurrentStage {
	case UpdateInitiated:
		result, err := HandleUpdateInitiated(ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	case ReplicaCreated:
		result, err := HandleReplicaCreated(r, ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	case ReplicaStarting:
		result, err := HandleReplicaStarting(r, ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	case ReplicaReady:
		result, err := HandleReplicaReady(ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	case MainShuttingDown:
		result, err := HandleMainShuttingDown(r, ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	case MainReady:
		result, err := HandleMainReady(r, ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	case UpdateFinished:
		result, err := HandleUpdateFinished(ctx, checkpoint)
		if err != nil {
			return false, err
		}
		return result, nil
	default:
		panic("unhandled default case")
	}
}

func HandleUpdateInitiated(ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	err := checkpoint.DoCheckpointWithDesiredStage(ctx, ReplicaCreated)
	if err != nil {
		return false, err
	}
	return true, nil
}
func HandleReplicaCreated(r *TeamcityReconciler, ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	var mainStatefulSet v1.StatefulSet
	instance := checkpoint.Instance
	mainNodeNamespacedName := instance.Spec.MainNode.GetNamespacedNameFromNamespace(instance.Namespace)
	if err := r.Get(ctx, mainNodeNamespacedName, &mainStatefulSet); err != nil {
		return false, err
	}
	roStatefulSet := resource.BuildROStatefulSet(&instance)
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var apiError error
		_, apiError = controllerutil.CreateOrUpdate(ctx, r.Client, roStatefulSet, func() error {
			return resource.UpdateROStatefulSet(r.Scheme, &instance, &mainStatefulSet, roStatefulSet)
		})
		return apiError
	})
	err = checkpoint.DoCheckpointWithDesiredStage(ctx, ReplicaStarting)
	if err != nil {
		return false, err
	}
	return true, nil
}
func HandleReplicaStarting(r *TeamcityReconciler, ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	instance := checkpoint.Instance
	roStatefulSetName := resource.GetROStatefulSetNamespacedName(&instance)
	var roStatefulSet v1.StatefulSet
	if err := r.Get(ctx, roStatefulSetName, &roStatefulSet); err != nil {
		return false, err
	}
	if roStatefulSet.Status.AvailableReplicas > 0 {
		err := checkpoint.DoCheckpointWithDesiredStage(ctx, ReplicaReady)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}
func HandleReplicaReady(ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	err := checkpoint.DoCheckpointWithDesiredStage(ctx, MainShuttingDown)
	if err != nil {
		return false, err
	}
	return false, nil
}
func HandleMainShuttingDown(r *TeamcityReconciler, ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	instance := checkpoint.Instance
	mainStatefulSetNamespacedName := instance.Spec.MainNode.GetNamespacedNameFromNamespace(instance.Namespace)

	mainNodeUpdateFinished, err := isNodeUpdateFinished(r, ctx, mainStatefulSetNamespacedName)
	if err != nil {
		return false, err
	}
	isMainNodeStatefulSetNewestGeneration, err := isNewestGeneration(r, ctx, mainStatefulSetNamespacedName)
	if err != nil {
		return false, err
	}

	if mainNodeUpdateFinished && isMainNodeStatefulSetNewestGeneration {
		err := checkpoint.DoCheckpointWithDesiredStage(ctx, MainReady)
		if err != nil {
			return false, err
		}
	}
	return true, nil

}
func HandleMainReady(r *TeamcityReconciler, ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	instance := checkpoint.Instance
	roStatefulSetName := resource.GetROStatefulSetNamespacedName(&instance)
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

	err := checkpoint.DoCheckpointWithDesiredStage(ctx, UpdateFinished)
	if err != nil {
		return false, err
	}
	return true, nil
}
func HandleUpdateFinished(ctx context.Context, checkpoint *Checkpoint) (bool, error) {
	err := checkpoint.Delete(ctx)
	if err != nil {
		return false, err
	}
	return false, nil
}
