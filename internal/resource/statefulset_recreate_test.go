package resource

import (
	"testing"

	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestImmutableStatefulSetSpecChanged(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		desired  string
		changed  bool
	}{
		{name: "unchanged empty serviceName", existing: "", desired: "", changed: false},
		{name: "unchanged serviceName", existing: "svc-a", desired: "svc-a", changed: false},
		{name: "serviceName added", existing: "", desired: "svc-a", changed: true},
		{name: "serviceName changed", existing: "svc-a", desired: "svc-b", changed: true},
		{name: "serviceName removed", existing: "svc-a", desired: "", changed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := &v1.StatefulSet{Spec: v1.StatefulSetSpec{ServiceName: tt.existing}}
			desired := &v1.StatefulSet{Spec: v1.StatefulSetSpec{ServiceName: tt.desired}}

			if got := ImmutableStatefulSetSpecChanged(existing, desired); got != tt.changed {
				t.Fatalf("ImmutableStatefulSetSpecChanged() = %v, want %v", got, tt.changed)
			}
		})
	}
}

func TestFormatImmutableStatefulSetFieldChanges(t *testing.T) {
	changes := []ImmutableStatefulSetFieldChange{
		{Field: "spec.serviceName", Current: "(not set)", Desired: "headless-svc"},
	}

	got := FormatImmutableStatefulSetFieldChanges(changes)
	want := "spec.serviceName: current=(not set), desired=headless-svc"
	if got != want {
		t.Fatalf("FormatImmutableStatefulSetFieldChanges() = %q, want %q", got, want)
	}
}

func TestBuildDesiredStatefulSetSetsServiceName(t *testing.T) {
	instance := &TeamCity{
		Spec: TeamCitySpec{
			Image: "jetbrains/teamcity-server",
			DataDirVolumeClaim: CustomPersistentVolumeClaim{
				Name: "data",
				VolumeMount: corev1.VolumeMount{
					Name:      "data",
					MountPath: "/storage",
				},
			},
			MainNode: Node{
				Name: "main-node",
				Spec: NodeSpec{
					ServiceName: "headless-svc",
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("500m"),
						"memory": resource.MustParse("1Gi"),
					},
				},
			},
		},
	}

	labels := map[string]string{"app.kubernetes.io/name": instance.Name}
	desired := BuildDesiredStatefulSet(instance, instance.Spec.MainNode, labels)

	if desired.Spec.ServiceName != "headless-svc" {
		t.Fatalf("expected serviceName headless-svc, got %q", desired.Spec.ServiceName)
	}
}
