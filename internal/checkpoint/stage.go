package checkpoint

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Stage int64

const (
	UpdateInitiated Stage = iota
	ReplicaCreated
	ReplicaStarting
	ReplicaReady
	MainShuttingDown
	MainReady
	UpdateFinished
	StageConfigMapKey        string = "stage"
	StageConfigMapNamePrefix string = "update-checkpoint"
)

const (
	StageUpdateInitiated  = "update-initiated"
	StageReplicaCreated   = "replica-created"
	StageReplicaStarting  = "replica-starting"
	StageReplicaReady     = "replica-ready"
	StageMainShuttingDown = "main-shutting-down"
	StageMainReady        = "main-ready"
	StageUpdateFinished   = "update-finished"
)

func NewStage(stage string) Stage {
	switch stage {
	case StageReplicaCreated:
		return ReplicaCreated
	case StageReplicaStarting:
		return ReplicaStarting
	case StageReplicaReady:
		return ReplicaReady
	case StageMainReady:
		return MainReady
	case StageMainShuttingDown:
		return MainShuttingDown
	case StageUpdateFinished:
		return UpdateFinished
	default:
		return UpdateInitiated
	}
}

func (s Stage) String() string {
	switch s {
	case ReplicaCreated:
		return StageReplicaCreated
	case ReplicaStarting:
		return StageReplicaStarting
	case ReplicaReady:
		return StageReplicaReady
	case MainReady:
		return StageMainReady
	case MainShuttingDown:
		return StageMainShuttingDown
	case UpdateFinished:
		return StageUpdateFinished
	default:
		return StageUpdateInitiated
	}
}

func (s Stage) BuildCheckpointConfigMap(instanceName string, instanceNamespace string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConstructCheckpointName(instanceName),
			Namespace: instanceNamespace,
		},
		Data: map[string]string{
			StageConfigMapKey: s.String(),
		},
	}
}

func (s Stage) canChangeStageValue(desired Stage) (bool, error) {
	if desired < s || desired-s > 1 {
		return false, fmt.Errorf("illegal stage transition: current stage '%s', desired stage '%s', difference must be 0 or 1",
			s, desired)
	}
	return true, nil
}

func ConstructCheckpointName(instanceName string) string {
	return fmt.Sprintf("%s-%s", StageConfigMapNamePrefix, instanceName)
}

func GetStageStringValueFromConfigMap(configMap *corev1.ConfigMap) (Stage, error) {
	stageStr, ok := configMap.Data[StageConfigMapKey]
	if !ok {
		return -1, fmt.Errorf("checkpoint ConfigMap is missing %s key", StageConfigMapKey)
	}
	return NewStage(stageStr), nil
}
