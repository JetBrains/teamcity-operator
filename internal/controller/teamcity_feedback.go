package controller

import (
	"context"
	"fmt"
	"strings"

	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	eventReasonStatefulSetRecreateBlocked = "StatefulSetRecreateBlocked"
	eventReasonStatefulSetRecreating      = "StatefulSetRecreating"
)

type StatefulSetRecreateBlockedError struct {
	StatefulSetName string
	NodeName        string
	Changes         []resource.ImmutableStatefulSetFieldChange
}

func newStatefulSetRecreateBlockedError(statefulSetName, nodeName string, changes []resource.ImmutableStatefulSetFieldChange) *StatefulSetRecreateBlockedError {
	return &StatefulSetRecreateBlockedError{
		StatefulSetName: statefulSetName,
		NodeName:        nodeName,
		Changes:         changes,
	}
}

func (e *StatefulSetRecreateBlockedError) Error() string {
	return e.UserMessage()
}

func (e *StatefulSetRecreateBlockedError) UserMessage() string {
	changeSummary := resource.FormatImmutableStatefulSetFieldChanges(e.Changes)
	if changeSummary == "" {
		changeSummary = "one or more immutable StatefulSet fields changed"
	}

	return fmt.Sprintf(
		"Cannot update StatefulSet %q for node %q: %s. Kubernetes does not allow these fields to be changed after the StatefulSet is created. "+
			"To apply this change, add annotation %s=%q to the TeamCity resource and reconcile again; "+
			"the operator will delete and recreate the StatefulSet, which restarts the TeamCity node.",
		e.StatefulSetName,
		e.NodeName,
		changeSummary,
		AllowStsRecreateAnnotationKey,
		AllowStsRecreateAnnotationValue,
	)
}

func (r *TeamcityReconciler) reportRecreateBlocked(
	ctx context.Context,
	instance *TeamCity,
	blocked *StatefulSetRecreateBlockedError,
) {
	logger := log.FromContext(ctx)
	message := blocked.UserMessage()

	logger.Info(
		"StatefulSet update blocked until recreate is explicitly allowed",
		"teamcity", instance.Name,
		"namespace", instance.Namespace,
		"statefulSet", blocked.StatefulSetName,
		"node", blocked.NodeName,
		"changes", resource.FormatImmutableStatefulSetFieldChanges(blocked.Changes),
		"requiredAnnotation", AllowStsRecreateAnnotationKey,
	)

	if err := updateTeamCityObjectStatusE(r, ctx, types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}, TEAMCITY_CRD_OBJECT_ERROR_STATE, message); err != nil {
		logger.Error(err, "Failed to write TeamCity status for blocked StatefulSet recreate")
	}

	if r.Recorder != nil {
		r.Recorder.Event(instance, v12.EventTypeWarning, eventReasonStatefulSetRecreateBlocked, message)
	}
}

func (r *TeamcityReconciler) reportRecreateInProgress(
	ctx context.Context,
	instance *TeamCity,
	statefulSetName string,
	changes []resource.ImmutableStatefulSetFieldChange,
) {
	logger := log.FromContext(ctx)
	message := fmt.Sprintf(
		"Recreating StatefulSet %q because immutable fields changed (%s)",
		statefulSetName,
		resource.FormatImmutableStatefulSetFieldChanges(changes),
	)

	logger.Info(
		"Recreating StatefulSet due to immutable field change",
		"teamcity", instance.Name,
		"namespace", instance.Namespace,
		"statefulSet", statefulSetName,
		"changes", resource.FormatImmutableStatefulSetFieldChanges(changes),
	)

	if err := updateTeamCityObjectStatusE(r, ctx, types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}, TEAMCITY_CRD_OBJECT_UPDATING_STATE, message); err != nil {
		logger.Error(err, "Failed to write TeamCity status for StatefulSet recreate")
	}

	if r.Recorder != nil {
		r.Recorder.Event(instance, v12.EventTypeNormal, eventReasonStatefulSetRecreating, message)
	}
}

func isStatefulSetImmutableFieldUpdateError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "Forbidden") &&
		strings.Contains(message, "statefulset spec for fields other than")
}
