package checkpoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewStage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Stage
	}{
		{"replica-created", StageReplicaCreated, ReplicaCreated},
		{"replica-starting", StageReplicaStarting, ReplicaStarting},
		{"replica-ready", StageReplicaReady, ReplicaReady},
		{"main-ready", StageMainReady, MainReady},
		{"main-shutting-down", StageMainShuttingDown, MainShuttingDown},
		{"update-finished", StageUpdateFinished, UpdateFinished},
		{"update-initiated", StageUpdateInitiated, UpdateInitiated},
		{"unknown defaults to update-initiated", "unknown", UpdateInitiated},
		{"empty defaults to update-initiated", "", UpdateInitiated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewStage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStageString(t *testing.T) {
	tests := []struct {
		name     string
		stage    Stage
		expected string
	}{
		{"UpdateInitiated", UpdateInitiated, StageUpdateInitiated},
		{"ReplicaCreated", ReplicaCreated, StageReplicaCreated},
		{"ReplicaStarting", ReplicaStarting, StageReplicaStarting},
		{"ReplicaReady", ReplicaReady, StageReplicaReady},
		{"MainShuttingDown", MainShuttingDown, StageMainShuttingDown},
		{"MainReady", MainReady, StageMainReady},
		{"UpdateFinished", UpdateFinished, StageUpdateFinished},
		{"unknown stage defaults to update-initiated", Stage(999), StageUpdateInitiated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stage.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCheckpointConfigMap(t *testing.T) {
	stage := ReplicaReady

	configMap := stage.BuildCheckpointConfigMap("my-teamcity", "default")

	assert.Equal(t, "update-checkpoint-my-teamcity", configMap.Name)
	assert.Equal(t, "default", configMap.Namespace)
	assert.Equal(t, StageReplicaReady, configMap.Data[StageConfigMapKey])
}

func TestCanChangeStageValue(t *testing.T) {
	tests := []struct {
		name        string
		current     Stage
		desired     Stage
		canChange   bool
		expectError bool
	}{
		{"same stage is allowed", ReplicaCreated, ReplicaCreated, true, false},
		{"next stage is allowed", ReplicaCreated, ReplicaStarting, true, false},
		{"skip one stage is not allowed", ReplicaCreated, ReplicaReady, false, true},
		{"going backwards is not allowed", ReplicaReady, ReplicaCreated, false, true},
		{"from start to next", UpdateInitiated, ReplicaCreated, true, false},
		{"from second to last to last", MainReady, UpdateFinished, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canChange, err := tt.current.canChangeStageValue(tt.desired)
			assert.Equal(t, tt.canChange, canChange)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConstructCheckpointName(t *testing.T) {
	result := ConstructCheckpointName("my-instance")
	assert.Equal(t, "update-checkpoint-my-instance", result)
}

func TestGetStageStringValueFromConfigMap(t *testing.T) {
	t.Run("returns stage when key exists", func(t *testing.T) {
		configMap := &corev1.ConfigMap{
			Data: map[string]string{
				StageConfigMapKey: StageReplicaReady,
			},
		}

		stage, err := GetStageStringValueFromConfigMap(configMap)

		assert.NoError(t, err)
		assert.Equal(t, ReplicaReady, stage)
	})

	t.Run("returns error when key is missing", func(t *testing.T) {
		configMap := &corev1.ConfigMap{
			Data: map[string]string{},
		}

		_, err := GetStageStringValueFromConfigMap(configMap)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})

	t.Run("returns error when data is nil", func(t *testing.T) {
		configMap := &corev1.ConfigMap{}

		_, err := GetStageStringValueFromConfigMap(configMap)

		assert.Error(t, err)
	})
}
