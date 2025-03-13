package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/checkpoint"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

func HandleStageChange(r *TeamcityReconciler, ctx context.Context, instance *TeamCity, currentStage checkpoint.Stage) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	switch currentStage {
	case checkpoint.Unknown:
		log.V(1).Info("Current update stage is  unknown")
		result, err := HandleUnknown(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	case checkpoint.UpdateStarted:
		log.V(1).Info("Current update stage is  update-started")
		result, err := HandleUpdateStarted(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	case checkpoint.ReplicaStarting:
		log.V(1).Info("Current update stage is  replica-starting")
		result, err := HandleReplicaStarting(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	case checkpoint.ReplicaReady:
		log.V(1).Info("Current update stage is  replica-ready")
		result, err := HandleReplicaReady(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	case checkpoint.MainShuttingDown:
		log.V(1).Info("Current update stage is  main-shutting-down")
		result, err := HandleMainShuttingDown(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	case checkpoint.MainReady:
		log.V(1).Info("Current update stage is  main-ready")
		result, err := HandleMainReady(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	case checkpoint.UpdateFinished:
		log.V(1).Info("Current update stage is  update-finished")
		result, err := HandleUpdateFinished(r, ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return result, nil
	}
	return ctrl.Result{}, nil
}

func HandleUnknown(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	err := DoCheckpointE(r, ctx, instance, checkpoint.UpdateStarted)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}
func HandleUpdateStarted(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	//create RO node and apply it
	var mainStatefulSet v1.StatefulSet
	if err := r.Get(ctx, GetMainStatefulSetNamespacedName(instance), &mainStatefulSet); err != nil {
		return ctrl.Result{}, err
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
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(reconciliationRequeueInterval)}, nil
}
func HandleReplicaStarting(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	roStatefulSetName := resource.GetROStatefulSetNamespacedName(instance)
	var roStatefulSet v1.StatefulSet
	if err := r.Get(ctx, roStatefulSetName, &roStatefulSet); err != nil {
		return ctrl.Result{}, err
	}
	if roStatefulSet.Status.AvailableReplicas > 0 {
		err := DoCheckpointE(r, ctx, instance, checkpoint.ReplicaReady)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(reconciliationRequeueInterval)}, nil
}
func HandleReplicaReady(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	err := DoCheckpointE(r, ctx, instance, checkpoint.MainShuttingDown)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}
func HandleMainShuttingDown(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	mainStatefulSetName := GetMainStatefulSetNamespacedName(instance)
	var mainStatefulSet v1.StatefulSet
	if err := r.Get(ctx, mainStatefulSetName, &mainStatefulSet); err != nil {
		return ctrl.Result{}, err
	}

	//check if we are looking at the latest generation of STS and check its available replicas
	if mainStatefulSet.Status.AvailableReplicas > 0 && mainStatefulSet.Generation == mainStatefulSet.Status.ObservedGeneration {
		err := DoCheckpointE(r, ctx, instance, checkpoint.MainReady)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(reconciliationRequeueInterval)}, nil

}
func HandleMainReady(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	roStatefulSetName := resource.GetROStatefulSetNamespacedName(instance)
	var roStatefulSet v1.StatefulSet
	roExists := true
	if err := r.Get(ctx, roStatefulSetName, &roStatefulSet); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			roExists = false
		}
	}
	if roExists {
		if err := r.Delete(ctx, &roStatefulSet); err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
	}

	err := DoCheckpointE(r, ctx, instance, checkpoint.UpdateFinished)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(reconciliationRequeueInterval)}, nil
}
func HandleUpdateFinished(r *TeamcityReconciler, ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	err := DeleteCheckPoint(r, ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
