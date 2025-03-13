package checkpoint

import "fmt"

type Stage int64

const (
	Unknown Stage = iota
	ReplicaCreated
	ReplicaStarting
	ReplicaReady
	MainShuttingDown
	MainReady
	UpdateFinished
)

const (
	StageUnknown          = "unknown"
	StageReplicaCreated   = "replica-created"
	StageReplicaStarting  = "replica-starting"
	StageReplicaReady     = "replica-ready"
	StageMainShuttingDown = "main-shutting-down"
	StageMainReady        = "main-ready"
	StageUpdateFinished   = "update-finished"
)

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
		return StageUnknown
	}
}

func ParseStage(stageStr string) (Stage, error) {
	switch stageStr {
	case StageReplicaCreated:
		return ReplicaCreated, nil
	case StageReplicaStarting:
		return ReplicaStarting, nil
	case StageReplicaReady:
		return ReplicaReady, nil
	case StageMainReady:
		return MainReady, nil
	case StageMainShuttingDown:
		return MainShuttingDown, nil
	case StageUpdateFinished:
		return UpdateFinished, nil
	default:
		return Unknown, fmt.Errorf("invalid stage string: %s", stageStr)
	}
}
