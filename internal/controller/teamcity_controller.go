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
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	jetbrainscomv1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
)

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

	log.Info(fmt.Sprintf("Reconciling object: %s.", req.Name))

	var teamcity jetbrainscomv1alpha1.TeamCity
	if err := r.Get(ctx, req.NamespacedName, &teamcity); err != nil {
		log.Error(err, "unable to fetch TeamCity")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info(fmt.Sprintf("Found object %s in namespace %s.", teamcity.Name, teamcity.Namespace))

	desiredStatefulSet := &v1.StatefulSet{}
	desiredStatefulSet.Name = req.Name
	desiredStatefulSet.Namespace = req.Namespace
	desiredStatefulSet.Spec.Template.Spec.Containers = []v12.Container{}
	desiredStatefulSet.Spec.Template.Spec.Containers = append(desiredStatefulSet.Spec.Template.Spec.Containers, v12.Container{})
	desiredStatefulSet.Spec.Template.Spec.Containers[0].Image = teamcity.Spec.Image
	desiredStatefulSet.Spec.Replicas = &teamcity.Spec.Replicas
	if err := controllerutil.SetControllerReference(&teamcity, desiredStatefulSet, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundStatefulSet := &v1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: desiredStatefulSet.Name, Namespace: desiredStatefulSet.Namespace}, foundStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating StatefulSet", "StatefulSet", desiredStatefulSet.Name)
		err = r.Create(ctx, desiredStatefulSet)
	} else if err == nil {
		if foundStatefulSet.Spec.Template.Spec.Containers[0].Image != desiredStatefulSet.Spec.Template.Spec.Containers[0].Image {
			foundStatefulSet.Spec.Template.Spec.Containers[0].Image = desiredStatefulSet.Spec.Template.Spec.Containers[0].Image
		}
		if foundStatefulSet.Spec.Replicas != desiredStatefulSet.Spec.Replicas {
			foundStatefulSet.Spec.Replicas = desiredStatefulSet.Spec.Replicas
		}
		log.V(1).Info("Updating StatefulSet", "StatefulSet", desiredStatefulSet.Name)
		err = r.Update(ctx, foundStatefulSet)

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamcityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jetbrainscomv1alpha1.TeamCity{}).
		Owns(&v1.StatefulSet{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
