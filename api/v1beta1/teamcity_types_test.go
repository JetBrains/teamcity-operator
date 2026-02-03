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

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNamespacedNameFromNamespace(t *testing.T) {
	node := Node{
		Name: "test-node",
	}

	result := node.GetNamespacedNameFromNamespace("test-namespace")

	assert.Equal(t, "test-node", result.Name)
	assert.Equal(t, "test-namespace", result.Namespace)
}

func TestGetAllNodes(t *testing.T) {
	instance := TeamCity{
		Spec: TeamCitySpec{
			MainNode: Node{Name: "main"},
			SecondaryNodes: []Node{
				{Name: "secondary-1"},
				{Name: "secondary-2"},
			},
		},
	}

	result := instance.GetAllNodes()

	assert.Len(t, result, 3)
	// SecondaryNodes are appended first, then MainNode
	assert.Equal(t, "secondary-1", result[0].Name)
	assert.Equal(t, "secondary-2", result[1].Name)
	assert.Equal(t, "main", result[2].Name)
}

func TestGetAllNodesWithNoSecondary(t *testing.T) {
	instance := TeamCity{
		Spec: TeamCitySpec{
			MainNode: Node{Name: "main"},
		},
	}

	result := instance.GetAllNodes()

	assert.Len(t, result, 1)
	assert.Equal(t, "main", result[0].Name)
}

func TestStartUpPropertiesConfigProvided(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]string
		expected bool
	}{
		{
			name:     "returns true when startup properties are provided",
			config:   map[string]string{"key": "value"},
			expected: true,
		},
		{
			name:     "returns false when startup properties are empty",
			config:   map[string]string{},
			expected: false,
		},
		{
			name:     "returns false when startup properties are nil",
			config:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := TeamCity{
				Spec: TeamCitySpec{
					StartupPropertiesConfig: tt.config,
				},
			}
			assert.Equal(t, tt.expected, instance.StartUpPropertiesConfigProvided())
		})
	}
}

func TestDatabaseSecretProvided(t *testing.T) {
	tests := []struct {
		name     string
		secret   DatabaseSecret
		expected bool
	}{
		{
			name:     "returns true when database secret is provided",
			secret:   DatabaseSecret{Secret: "my-secret"},
			expected: true,
		},
		{
			name:     "returns false when database secret is empty",
			secret:   DatabaseSecret{Secret: ""},
			expected: false,
		},
		{
			name:     "returns false when database secret is not set",
			secret:   DatabaseSecret{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := TeamCity{
				Spec: TeamCitySpec{
					DatabaseSecret: tt.secret,
				},
			}
			assert.Equal(t, tt.expected, instance.DatabaseSecretProvided())
		})
	}
}

func TestDataDirPath(t *testing.T) {
	instance := TeamCity{
		Spec: TeamCitySpec{
			DataDirVolumeClaim: CustomPersistentVolumeClaim{
				VolumeMount: corev1.VolumeMount{
					MountPath: "/data/teamcity",
				},
			},
		},
	}

	assert.Equal(t, "/data/teamcity", instance.DataDirPath())
}

func TestGetAllCustomPersistentVolumeClaim(t *testing.T) {
	dataDirPVC := CustomPersistentVolumeClaim{Name: "data-dir"}
	additionalPVC1 := CustomPersistentVolumeClaim{Name: "additional-1"}
	additionalPVC2 := CustomPersistentVolumeClaim{Name: "additional-2"}

	instance := TeamCity{
		Spec: TeamCitySpec{
			DataDirVolumeClaim: dataDirPVC,
			PersistentVolumeClaims: []CustomPersistentVolumeClaim{
				additionalPVC1,
				additionalPVC2,
			},
		},
	}

	result := instance.GetAllCustomPersistentVolumeClaim()

	assert.Len(t, result, 3)
	assert.Equal(t, "additional-1", result[0].Name)
	assert.Equal(t, "additional-2", result[1].Name)
	assert.Equal(t, "data-dir", result[2].Name)
}

func TestServiceAccountProvided(t *testing.T) {
	tests := []struct {
		name           string
		serviceAccount ServiceAccount
		expected       bool
	}{
		{
			name:           "returns true when service account name is provided",
			serviceAccount: ServiceAccount{Name: "my-sa"},
			expected:       true,
		},
		{
			name:           "returns false when service account name is empty",
			serviceAccount: ServiceAccount{Name: ""},
			expected:       false,
		},
		{
			name:           "returns false when service account is not set",
			serviceAccount: ServiceAccount{},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := TeamCity{
				Spec: TeamCitySpec{
					ServiceAccount: tt.serviceAccount,
				},
			}
			assert.Equal(t, tt.expected, instance.ServiceAccountProvided())
		})
	}
}

func TestIsMultiNode(t *testing.T) {
	tests := []struct {
		name           string
		secondaryNodes []Node
		expected       bool
	}{
		{
			name:           "returns true when secondary nodes are present",
			secondaryNodes: []Node{{Name: "secondary-1"}},
			expected:       true,
		},
		{
			name:           "returns true when multiple secondary nodes are present",
			secondaryNodes: []Node{{Name: "secondary-1"}, {Name: "secondary-2"}},
			expected:       true,
		},
		{
			name:           "returns false when no secondary nodes",
			secondaryNodes: []Node{},
			expected:       false,
		},
		{
			name:           "returns false when secondary nodes is nil",
			secondaryNodes: nil,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := TeamCity{
				Spec: TeamCitySpec{
					SecondaryNodes: tt.secondaryNodes,
				},
			}
			assert.Equal(t, tt.expected, instance.IsMultiNode())
		})
	}
}

func TestUsesZeroDownTimeUpgradePolicy(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name: "returns true when zero-downtime annotation is set",
			annotations: map[string]string{
				UpdatePolicyAnnotationKey: ZeroDownTimeAnnotation,
			},
			expected: true,
		},
		{
			name: "returns false when annotation has different value",
			annotations: map[string]string{
				UpdatePolicyAnnotationKey: "rolling",
			},
			expected: false,
		},
		{
			name:        "returns false when annotation is not set",
			annotations: map[string]string{},
			expected:    false,
		},
		{
			name:        "returns false when annotations is nil",
			annotations: nil,
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
			assert.Equal(t, tt.expected, instance.UsesZeroDownTimeUpgradePolicy())
		})
	}
}
