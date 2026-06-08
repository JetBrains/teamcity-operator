package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAllowsStatefulSetRecreate(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name: "returns true when recreate annotation is set",
			annotations: map[string]string{
				AllowStsRecreateAnnotationKey: AllowStsRecreateAnnotationValue,
			},
			expected: true,
		},
		{
			name: "returns false when recreate annotation has different value",
			annotations: map[string]string{
				AllowStsRecreateAnnotationKey: "false",
			},
			expected: false,
		},
		{
			name:        "returns false when recreate annotation is missing",
			annotations: map[string]string{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}
			assert.Equal(t, tt.expected, instance.AllowsStatefulSetRecreate())
		})
	}
}

func TestServiceNameChangedInSpec(t *testing.T) {
	base := TeamCity{
		Spec: TeamCitySpec{
			MainNode: Node{
				Name: "main",
				Spec: NodeSpec{ServiceName: "svc-a"},
			},
			SecondaryNodes: []Node{
				{Name: "secondary", Spec: NodeSpec{ServiceName: "svc-secondary"}},
			},
		},
	}

	tests := []struct {
		name     string
		updated  TeamCity
		expected bool
	}{
		{
			name:     "no serviceName changes",
			updated:  base,
			expected: false,
		},
		{
			name: "main node serviceName changed",
			updated: TeamCity{
				Spec: TeamCitySpec{
					MainNode: Node{Name: "main", Spec: NodeSpec{ServiceName: "svc-b"}},
					SecondaryNodes: []Node{
						{Name: "secondary", Spec: NodeSpec{ServiceName: "svc-secondary"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "secondary node serviceName changed",
			updated: TeamCity{
				Spec: TeamCitySpec{
					MainNode: Node{Name: "main", Spec: NodeSpec{ServiceName: "svc-a"}},
					SecondaryNodes: []Node{
						{Name: "secondary", Spec: NodeSpec{ServiceName: "svc-new"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "new secondary node does not count as change",
			updated: TeamCity{
				Spec: TeamCitySpec{
					MainNode: Node{Name: "main", Spec: NodeSpec{ServiceName: "svc-a"}},
					SecondaryNodes: []Node{
						{Name: "secondary", Spec: NodeSpec{ServiceName: "svc-secondary"}},
						{Name: "secondary-2", Spec: NodeSpec{ServiceName: "svc-new"}},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ServiceNameChangedInSpec(&base, &tt.updated))
		})
	}
}
