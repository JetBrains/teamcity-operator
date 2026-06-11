package controller

import (
	"context"
	"fmt"
	"time"

	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	statefulSetRecreateRequeueInterval = 2 * time.Second
)

func isStatefulSetBuilder(builder resource.ResourceBuilder) bool {
	switch builder.(type) {
	case *resource.StatefulSetBuilder, *resource.SecondaryStatefulSetBuilder:
		return true
	default:
		return false
	}
}

func (r *TeamcityReconciler) reconcileStatefulSetBeforeCreateOrUpdate(
	ctx context.Context,
	instance *TeamCity,
	builder resource.ResourceBuilder,
	object client.Object,
) (ctrl.Result, error) {
	if !isStatefulSetBuilder(builder) {
		return ctrl.Result{}, nil
	}

	node, ok, err := nodeForStatefulSetObject(instance, builder, object)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ok {
		return ctrl.Result{}, fmt.Errorf("unable to resolve node for StatefulSet %q", object.GetName())
	}

	role := statefulSetRoleForBuilder(builder)
	labels := metadata.GetStatefulSetLabels(instance.Name, node.Name, role, instance.Labels)
	desired := resource.BuildDesiredStatefulSet(instance, node, labels)

	existing := &v1.StatefulSet{}
	namespacedName := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      object.GetName(),
	}
	if err := r.Get(ctx, namespacedName, existing); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	changes := resource.GetImmutableStatefulSetFieldChanges(existing, desired)
	if len(changes) == 0 {
		return ctrl.Result{}, nil
	}

	if !instance.AllowsStatefulSetRecreate() {
		return ctrl.Result{}, newStatefulSetRecreateBlockedError(existing.Name, node.Name, changes)
	}

	r.reportRecreateInProgress(ctx, instance, existing.Name, changes)

	foreground := metav1.DeletePropagationForeground
	if err := r.Delete(ctx, existing, &client.DeleteOptions{PropagationPolicy: &foreground}); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.Get(ctx, namespacedName, existing); err == nil {
		return ctrl.Result{Requeue: true, RequeueAfter: statefulSetRecreateRequeueInterval}, nil
	}
	if !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func nodeForStatefulSetObject(instance *TeamCity, builder resource.ResourceBuilder, object client.Object) (Node, bool, error) {
	switch builder.(type) {
	case *resource.StatefulSetBuilder:
		if object.GetName() != instance.Spec.MainNode.Name {
			return Node{}, false, nil
		}
		return instance.Spec.MainNode, true, nil
	case *resource.SecondaryStatefulSetBuilder:
		for _, node := range instance.Spec.SecondaryNodes {
			if node.Name == object.GetName() {
				return node, true, nil
			}
		}
		return Node{}, false, fmt.Errorf("secondary node %q not found in TeamCity spec", object.GetName())
	default:
		return Node{}, false, nil
	}
}

func statefulSetRoleForBuilder(builder resource.ResourceBuilder) string {
	switch builder.(type) {
	case *resource.StatefulSetBuilder:
		return "main"
	case *resource.SecondaryStatefulSetBuilder:
		return "secondary"
	default:
		return ""
	}
}

func (r *TeamcityReconciler) statefulSetRecreateBlockedFromAPIError(
	ctx context.Context,
	instance *TeamCity,
	builder resource.ResourceBuilder,
	object client.Object,
	apiErr error,
) (*StatefulSetRecreateBlockedError, bool) {
	if !isStatefulSetBuilder(builder) || !isStatefulSetImmutableFieldUpdateError(apiErr) {
		return nil, false
	}

	node, ok, err := nodeForStatefulSetObject(instance, builder, object)
	if err != nil || !ok {
		return nil, false
	}

	role := statefulSetRoleForBuilder(builder)
	labels := metadata.GetStatefulSetLabels(instance.Name, node.Name, role, instance.Labels)
	desired := resource.BuildDesiredStatefulSet(instance, node, labels)

	existing := &v1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      object.GetName(),
	}, existing); err != nil {
		return nil, false
	}

	changes := resource.GetImmutableStatefulSetFieldChanges(existing, desired)
	if len(changes) == 0 {
		changes = []resource.ImmutableStatefulSetFieldChange{
			{
				Field:   "spec (immutable StatefulSet field)",
				Current: "(cluster value)",
				Desired: "(requested value)",
			},
		}
	}

	return newStatefulSetRecreateBlockedError(existing.Name, node.Name, changes), true
}
