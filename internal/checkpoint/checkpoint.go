package checkpoint

import (
	"context"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Checkpoint struct {
	Client       client.Client
	CurrentStage Stage
	Instance     TeamCity
}

func NewCheckpoint(client client.Client, instance TeamCity, currentStage string) *Checkpoint {
	return &Checkpoint{
		Client:       client,
		Instance:     instance,
		CurrentStage: NewStage(currentStage),
	}
}

func (c *Checkpoint) DoCheckpointWithDesiredStage(ctx context.Context, desiredStage Stage) error {
	currentStage, err := c.FetchCurrentStageFromCluster(ctx)
	if err != nil {
		if errors.IsNotFound(err) { // if checkpoint CM does not exist we need to set an initial value and create a new CM
			initialStage := c.getInitialStageFromInstance()
			c.CurrentStage = initialStage
			err = c.Create(ctx)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	c.CurrentStage = currentStage
	canChangeStage, err := c.CurrentStage.canChangeStageValue(desiredStage)
	if err != nil {
		return err
	}
	if canChangeStage {
		c.CurrentStage = desiredStage
		err = c.Update(ctx)
	}
	return nil
}

func (c *Checkpoint) Create(ctx context.Context) error {
	configMap := c.toConfigMapObject()
	if err := c.Client.Create(ctx, &configMap); err != nil {
		return err
	}
	return nil
}

func (c *Checkpoint) FetchCurrentStageFromCluster(ctx context.Context) (Stage, error) {
	configMap, err := c.GetConfigMap(ctx)
	if err != nil {
		return -1, err
	}
	stage, err := GetStageStringValueFromConfigMap(&configMap)
	if err != nil {
		return -1, err
	}
	return stage, nil
}

func (c *Checkpoint) GetConfigMap(ctx context.Context) (v1.ConfigMap, error) {
	var configMap v1.ConfigMap
	namespacedName := c.getNamespacedName()
	if err := c.Client.Get(ctx, namespacedName, &configMap); err != nil {
		if !errors.IsNotFound(err) {
			return configMap, err
		}
		return configMap, nil
	}
	return configMap, nil
}

func (c *Checkpoint) Update(ctx context.Context) error {
	checkpointCM := c.toConfigMapObject()
	if err := c.Client.Update(ctx, &checkpointCM); err != nil {
		return nil
	}
	return nil
}

func (c *Checkpoint) Delete(ctx context.Context) error {
	configMap, err := c.GetConfigMap(ctx)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	if err := c.Client.Delete(ctx, &configMap); err != nil {
		return err
	}

	return nil
}

func (c *Checkpoint) getCheckpointConfigMapName() string {
	return fmt.Sprintf("%s-%s", StageConfigMapNamePrefix, c.Instance.Name)
}

func (c *Checkpoint) getNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      c.getCheckpointConfigMapName(),
		Namespace: c.Instance.Namespace,
	}
}

func (c *Checkpoint) toConfigMapObject() v1.ConfigMap {
	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConstructCheckpointName(c.Instance.Name),
			Namespace: c.Instance.Namespace,
		},
		Data: map[string]string{
			StageConfigMapKey: c.CurrentStage.String(),
		},
	}
}

func (c *Checkpoint) getInitialStageFromInstance() Stage {
	if c.Instance.IsMultiNode() {
		return ReplicaReady
	}
	return UpdateInitiated
}
