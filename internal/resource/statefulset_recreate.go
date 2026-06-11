package resource

import (
	"fmt"
	"strings"

	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
)

const immutableFieldNotSet = "(not set)"

// ImmutableStatefulSetFieldChange describes one immutable StatefulSet field drift.
type ImmutableStatefulSetFieldChange struct {
	Field   string
	Current string
	Desired string
}

// ImmutableStatefulSetSpecChanged reports whether desired changes require deleting and
// recreating the StatefulSet because Kubernetes forbids in-place updates.
func ImmutableStatefulSetSpecChanged(existing, desired *v1.StatefulSet) bool {
	return len(GetImmutableStatefulSetFieldChanges(existing, desired)) > 0
}

// GetImmutableStatefulSetFieldChanges returns the immutable StatefulSet fields that differ.
func GetImmutableStatefulSetFieldChanges(existing, desired *v1.StatefulSet) []ImmutableStatefulSetFieldChange {
	var changes []ImmutableStatefulSetFieldChange

	if existing.Spec.ServiceName != desired.Spec.ServiceName {
		changes = append(changes, ImmutableStatefulSetFieldChange{
			Field:   "spec.serviceName",
			Current: displayStatefulSetFieldValue(existing.Spec.ServiceName),
			Desired: displayStatefulSetFieldValue(desired.Spec.ServiceName),
		})
	}

	return changes
}

// FormatImmutableStatefulSetFieldChanges renders field changes for logs, events, and status.
func FormatImmutableStatefulSetFieldChanges(changes []ImmutableStatefulSetFieldChange) string {
	parts := make([]string, 0, len(changes))
	for _, change := range changes {
		parts = append(parts, fmt.Sprintf("%s: current=%s, desired=%s", change.Field, change.Current, change.Desired))
	}
	return strings.Join(parts, "; ")
}

func displayStatefulSetFieldValue(value string) string {
	if value == "" {
		return immutableFieldNotSet
	}
	return value
}

// BuildDesiredStatefulSet materializes the StatefulSet spec the operator would apply for node.
func BuildDesiredStatefulSet(instance *TeamCity, node Node, labels map[string]string) *v1.StatefulSet {
	statefulSet := CreateEmptyStatefulSet(node.Name, instance.Namespace, labels)
	ConfigureStatefulSet(instance, node, &statefulSet)

	var container v12.Container
	ConfigureContainer(instance, node, &container)
	statefulSet.Spec.Template.Spec.Containers = []v12.Container{container}

	return &statefulSet
}
