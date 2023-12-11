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
	jetbrainscomv1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	jetbrainscomv1beta1 "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/predicate"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	"git.jetbrains.team/tch/teamcity-operator/internal/validator"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const teamcityFinalizer = "teamcity.jetbrains.com/finalizer"

// TeamcityReconciler reconciles a TeamCity object
type TeamcityReconciler struct {
	client.Client
	Clientset *kubernetes.Clientset
	Scheme    *runtime.Scheme
}

//+kubebuilder:rbac:groups=jetbrains.com,resources=teamcities,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jetbrains.com,resources=teamcities/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jetbrains.com,resources=teamcities/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

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
	var err error

	if teamcity, err = GetTeamCityObjectE(r, ctx, req.NamespacedName); err != nil {
		return ctrl.Result{}, err
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
		object, err := builder.Build()
		if err != nil {
			return ctrl.Result{}, err
		}

		// validate resources required to creation of object
		// depending on type of object we want to perform different checks
		// for example: for sts we need to make that database secret(if provided) is valid to run teamcity
		switch builder.(type) {

		case *resource.StatefulSetBuilder:
			if teamcity.Spec.DatabaseSecret.Secret != "" {
				if err = r.validateDatabaseSecret(ctx, req, teamcity.Spec.DatabaseSecret.Secret); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		var operationResult controllerutil.OperationResult
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			log.V(1).Info(fmt.Sprintf("Attempting to update object of type %s", object.GetObjectKind().GroupVersionKind().Kind))
			var apiError error
			operationResult, apiError = controllerutil.CreateOrUpdate(ctx, r.Client, object, func() error {
				return builder.Update(object)
			})
			return apiError
		})

		if err != nil {
			log.V(1).Error(err, "Failed to update object")
			return ctrl.Result{}, err
		}
		log.V(1).Info(fmt.Sprintf("Status of object %s is now %s", object.GetObjectKind().GroupVersionKind().Kind, operationResult))
	}
	_ = UpdateTeamCityObjectStatusE(r, ctx, req.NamespacedName, TEAMCITY_CRD_OBJECT_SUCCESS_STATE, "Successfully reconciled TeamCity")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamcityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jetbrainscomv1beta1.TeamCity{}, builder.WithPredicates(predicate.TeamcityEventPredicates())). //separate predicates for TC and STS as they should be handled differently
		Owns(&v1.StatefulSet{}, builder.WithPredicates(predicate.StatefulSetEventPredicates())).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func (r *TeamcityReconciler) finalizeTeamCity(log logr.Logger, teamcity *jetbrainscomv1alpha1.TeamCity) error {
	log.V(1).Info("Ran finalizers TeamCity object successfully")
	return nil
}

func (r *TeamcityReconciler) validateDatabaseSecret(ctx context.Context, req ctrl.Request, secretName string) (err error) {
	var databaseSecret v12.Secret
	if databaseSecret, err = GetSecretE(r, ctx, secretName, req.Namespace); err != nil {
		_ = UpdateTeamCityObjectStatusE(r, ctx, req.NamespacedName, TEAMCITY_CRD_OBJECT_ERROR_STATE, err.Error())
		return err
	}
	dbValidator := validator.DatabaseSecretValidator{Object: &databaseSecret}
	if err = dbValidator.ValidateObject(); err != nil {
		_ = UpdateTeamCityObjectStatusE(r, ctx, req.NamespacedName, TEAMCITY_CRD_OBJECT_ERROR_STATE, err.Error())
		return err
	}
	return err
}
