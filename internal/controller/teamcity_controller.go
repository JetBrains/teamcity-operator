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
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	"git.jetbrains.team/tch/teamcity-operator/internal/predicate"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"time"
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

	var teamcity TeamCity
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
		Client:   r.Client,
	}
	builders := resourceBuilder.ResourceBuilders()

	mainNodeRecreation, err := r.requiresMainNodeRecreation(ctx, &teamcity)

	if mainNodeRecreation {
		if _, err := r.reconcileROCreateOrUpdate(ctx, &teamcity, mainNodeRecreation); err != nil {
			return ctrl.Result{}, err
		}

		secondaryKey := types.NamespacedName{
			Name:      "update-ro-replica",
			Namespace: teamcity.Namespace,
		}
		secondaryNodeAvailable, _ := isNodeUpdateFinished(r, ctx, secondaryKey)

		if !secondaryNodeAvailable {
			return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(30000000000)}, nil
		}
	}

	for _, builder := range builders {
		if _, err := r.reconcileDelete(ctx, builder); err != nil {
			return ctrl.Result{}, err
		}

		if preconditionSuccess := r.validatePreconditions(ctx, builder, teamcity); !preconditionSuccess {
			log.V(1).Info("Preconditions are not satisfied")
			//we want to retry reconcile after preconditions will be met
			//RequeueAfter is specified in nanoseconds :melting_face
			return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(30000000000)}, nil
		}

		if _, err := r.reconcileCreateOrUpdate(ctx, builder); err != nil {
			return ctrl.Result{}, err
		}
	}

	if postconditionSuccess := r.validatePostcondition(ctx, teamcity); !postconditionSuccess {
		log.V(1).Info("Postconditions are not satisfied")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(30000000000)}, nil
	}

	if result, err := r.reconcileRODelete(ctx, &teamcity); err != nil {
		return result, err
	}

	_ = UpdateTeamCityObjectStatusE(r, ctx, req.NamespacedName, TEAMCITY_CRD_OBJECT_SUCCESS_STATE, "Successfully reconciled TeamCity")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamcityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&TeamCity{}, builder.WithPredicates(predicate.TeamcityEventPredicates())). //separate predicates for TC and STS as they should be handled differently
		Owns(&v1.StatefulSet{}, builder.WithPredicates(predicate.StatefulSetEventPredicates())).
		Owns(&v12.Service{}).
		Owns(&netv1.Ingress{}).
		Owns(&v12.PersistentVolumeClaim{}, builder.WithPredicates(predicate.PersistentVolumeClaimEventPredicates())).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func (r *TeamcityReconciler) finalizeTeamCity(log logr.Logger, teamcity *TeamCity) error {
	log.V(1).Info("Ran finalizers TeamCity object successfully")
	return nil
}

func (r *TeamcityReconciler) reconcileCreateOrUpdate(ctx context.Context, builder resource.ResourceBuilder) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	objectList, err := builder.BuildObjectList()
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, object := range objectList {
		var operationResult controllerutil.OperationResult
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var apiError error
			operationResult, apiError = controllerutil.CreateOrUpdate(ctx, r.Client, object, func() error {
				return builder.Update(object)
			})
			return apiError
		})

		if err != nil {
			log.V(1).Error(err, fmt.Sprintf("Failed to update object %s %s", object.GetObjectKind().GroupVersionKind().Kind, object.GetName()))
			return ctrl.Result{}, err
		}
		log.V(1).Info(fmt.Sprintf("Status of object %s %s is now %s", object.GetObjectKind().GroupVersionKind().Kind, object.GetName(), operationResult))

	}
	return ctrl.Result{}, nil
}

func (r *TeamcityReconciler) reconcileDelete(ctx context.Context, builder resource.ResourceBuilder) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	obsoleteObjects, err := builder.GetObsoleteObjects(ctx)
	if err != nil {
		log.V(1).Error(err, "Failed to get obsolete objects")
		return ctrl.Result{}, err
	}
	for _, object := range obsoleteObjects {
		// TODO: to check owner ref?
		if err := r.Delete(ctx, object); err != nil {
			log.V(1).Error(err, "Failed to delete obsolete object %s with type %s", object.GetName(), object.GetObjectKind().GroupVersionKind().Kind)
			return ctrl.Result{}, err
		}
		log.V(1).Info(fmt.Sprintf("Obsolete object %s %s was deleted", object.GetObjectKind().GroupVersionKind().Kind, object.GetName()))
	}
	return ctrl.Result{}, nil
}

func (r *TeamcityReconciler) validatePreconditions(ctx context.Context, builder resource.ResourceBuilder, instance TeamCity) (preconditionSuccessful bool) {
	log := log.FromContext(ctx)
	preconditionSuccessful = true

	switch builder.(type) {
	case *resource.SecondaryStatefulSetBuilder:
		if len(instance.Spec.SecondaryNodes) != 0 {
			log.V(1).Info("Checking if the main node has started before starting secondary nodes")
			mainNodeNamespacedName := types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Spec.MainNode.Name,
			}
			newestGeneration, err := isNewestGeneration(r, ctx, mainNodeNamespacedName)
			if err != nil {
				log.V(1).Error(err, "Unable to get generation information for the main node.")
			}

			updated, err := isNodeUpdateFinished(r, ctx, mainNodeNamespacedName)
			if err != nil {
				log.V(1).Error(err, "Unable to get revision status information of the main node")
			}

			log.V(1).Info(fmt.Sprintf("Newest generation: %s", strconv.FormatBool(newestGeneration)))
			log.V(1).Info(fmt.Sprintf("Main node updated: %s", strconv.FormatBool(updated)))
			preconditionSuccessful = newestGeneration && updated
		}
	}
	return preconditionSuccessful
}

func (r *TeamcityReconciler) requiresMainNodeRecreation(ctx context.Context, instance *TeamCity) (bool, error) {
	//log := log.FromContext(ctx)
	existing, err := GetStatefulSetByName(r, ctx, types.NamespacedName{Name: instance.Spec.MainNode.Name, Namespace: instance.Namespace})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	var desired v1.StatefulSet
	resource.ConfigureStatefulSet(instance, instance.Spec.MainNode, &desired)
	var container v12.Container
	resource.ConfigureContainer(instance, instance.Spec.MainNode, &container)
	desired.Spec.Template.Spec.Containers = []v12.Container{container}

	if !equality.Semantic.DeepDerivative(desired.Spec, existing.Spec) {
		return true, nil
	}
	return false, nil
}

func (r *TeamcityReconciler) createRoReplica(ctx context.Context, instance *TeamCity, main v1.StatefulSet) (client.ObjectKey, error) {
	//log := log.FromContext(ctx)
	secondaryNodeName := "update-ro-replica"
	secondrayNodeLabels := metadata.GetStatefulSetLabels(instance.Name, secondaryNodeName, "ro-replica", instance.Labels)
	secondary := v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secondaryNodeName,
			Namespace: instance.Namespace,
			Labels:    secondrayNodeLabels,
		},
	}
	secondary.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: secondrayNodeLabels,
	}
	secondary.Spec.Template.Spec = main.Spec.Template.Spec
	secondary.Spec.Template.Labels = secondrayNodeLabels

	secondaryNode := Node{
		Name: secondaryNodeName,
		Spec: NodeSpec{
			Requests: instance.Spec.MainNode.Spec.Requests,
		},
	}
	envVars := resource.BuildEnvVariablesFromGlobalAndNodeSpecificSettings(instance, secondaryNode)
	secondary.Spec.Template.Spec.Containers[0].Env = envVars
	if err := controllerutil.SetControllerReference(instance, &secondary, r.Scheme); err != nil {
		return client.ObjectKey{}, fmt.Errorf("failed setting controller reference: %w", err)
	}

	err := r.Create(ctx, &secondary)
	if err != nil {
		return client.ObjectKey{}, err
	}
	return client.ObjectKey{Name: secondary.Name, Namespace: secondary.Namespace}, nil
}

func (r *TeamcityReconciler) validatePostcondition(ctx context.Context, instance TeamCity) (conditionSuccessful bool) {
	log := log.FromContext(ctx)
	conditionSuccessful = true

	mainNodeNamespacedName := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Spec.MainNode.Name,
	}
	newestGeneration, err := isNewestGeneration(r, ctx, mainNodeNamespacedName)
	if err != nil {
		log.V(1).Error(err, "Unable to get generation information for the main node.")
	}

	updated, err := isNodeUpdateFinished(r, ctx, mainNodeNamespacedName)
	if err != nil {
		log.V(1).Error(err, "Unable to get revision status information of the main node")
	}

	log.V(1).Info(fmt.Sprintf("Newest generation: %s", strconv.FormatBool(newestGeneration)))
	log.V(1).Info(fmt.Sprintf("Main node updated: %s", strconv.FormatBool(updated)))
	conditionSuccessful = newestGeneration && updated

	return conditionSuccessful
}

func (r *TeamcityReconciler) reconcileROCreateOrUpdate(ctx context.Context, instance *TeamCity, mainNodeRecreationRequired bool) (ctrl.Result, error) {
	if !mainNodeRecreationRequired {
		return ctrl.Result{}, nil
	}
	mainStatefulset, err := GetStatefulSetByName(r, ctx, types.NamespacedName{Name: instance.Spec.MainNode.Name, Namespace: instance.Namespace})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	secondaryNodeName := "update-ro-replica"
	secondrayNodeLabels := metadata.GetStatefulSetLabels(instance.Name, secondaryNodeName, "ro-replica", instance.Labels)
	secondary := v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secondaryNodeName,
			Namespace: instance.Namespace,
			Labels:    secondrayNodeLabels,
		},
	}
	secondary.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: secondrayNodeLabels,
	}
	secondary.Spec.Template.Spec = mainStatefulset.Spec.Template.Spec
	secondary.Spec.Template.Labels = secondrayNodeLabels

	secondaryNode := Node{
		Name: secondaryNodeName,
		Spec: NodeSpec{
			Requests: instance.Spec.MainNode.Spec.Requests,
		},
	}
	envVars := resource.BuildEnvVariablesFromGlobalAndNodeSpecificSettings(instance, secondaryNode)
	secondary.Spec.Template.Spec.Containers[0].Env = envVars
	if err := controllerutil.SetControllerReference(instance, &secondary, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed setting controller reference: %w", err)
	}

	err = r.Create(ctx, &secondary)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *TeamcityReconciler) reconcileRODelete(ctx context.Context, instance *TeamCity) (ctrl.Result, error) {
	secondaryNodeName := "update-ro-replica"
	key := types.NamespacedName{
		Name:      secondaryNodeName,
		Namespace: instance.Namespace,
	}
	var secondary v1.StatefulSet
	if err := r.Get(ctx, key, &secondary); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.Delete(ctx, &secondary); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
