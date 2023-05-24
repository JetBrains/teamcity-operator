/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	jetbrainscomv1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
)

const teamcityFinalizer = "teamcity.jetbrains.com/finalizer"

// TeamcityReconciler reconciles a TeamCity object
type TeamcityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=jetbrains.com,resources=teamcities,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jetbrains.com,resources=teamcities/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jetbrains.com,resources=teamcities/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the TeamCity object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TeamcityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var teamcity jetbrainscomv1alpha1.TeamCity
	if err := r.Get(ctx, req.NamespacedName, &teamcity); err != nil {
		log.V(1).Info("Could not get TeamCity object. Ignoring")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	isMarkedForDeletion := teamcity.GetDeletionTimestamp() != nil
	if isMarkedForDeletion {
		log.V(1).Info("TeamCity object is marked for deletion")
		if err := r.finalizeTeamCity(log, &teamcity); err != nil {
			log.V(1).Error(err, "Failed to finalize TeamCity object")
			return ctrl.Result{}, err
		}
		log.V(1).Info("TeamCity object is finalized")
		controllerutil.RemoveFinalizer(&teamcity, teamcityFinalizer)
		log.V(1).Info("Finalizer is removed from TeamCity object")
		err := r.Update(ctx, &teamcity)
		if err != nil {
			log.V(1).Error(err, "Failed to update TeamCity object")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	resourceBuilder := resource.TeamCityResourceBuilder{
		Instance: &teamcity,
		Scheme:   r.Scheme,
	}
	builders := resourceBuilder.ResourceBuilders()

	for _, builder := range builders {
		resource, err := builder.Build()
		if err != nil {
			return ctrl.Result{}, err
		}
		operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, resource, func() error {
			return builder.Update(resource)
		})
		if err != nil {
			log.V(1).Error(err, "Failed to update resource")
			return ctrl.Result{}, err
		}
		log.V(1).Info(fmt.Sprintf("Reconcilation result for TeamCity object is %s", operationResult))

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamcityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jetbrainscomv1alpha1.TeamCity{}).
		Owns(&v1.StatefulSet{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func (r *TeamcityReconciler) finalizeTeamCity(log logr.Logger, teamcity *jetbrainscomv1alpha1.TeamCity) error {
	log.V(1).Info("Ran finalizers TeamCity object successfully")
	return nil
}
