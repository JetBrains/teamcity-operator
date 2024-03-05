package resource

import (
	"context"
	"fmt"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type StatefulSetBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) StatefulSet() *StatefulSetBuilder {
	return &StatefulSetBuilder{builder}
}

func (builder *StatefulSetBuilder) UpdateMayRequireStsRecreate() bool {
	return true
}

func (builder *StatefulSetBuilder) BuildObjectList() ([]client.Object, error) {
	mainNodeLabels := metadata.GetLabels(builder.Instance.Spec.MainNode.Name, builder.Instance.Labels)
	mainNode := CreateEmptyStatefulSet(builder.Instance.Spec.MainNode.Name, builder.Instance.Namespace, mainNodeLabels)
	return []client.Object{
		&mainNode,
	}, nil
}

func (builder *StatefulSetBuilder) Update(object client.Object) error {
	statefulSpec := object.(*v1.StatefulSet)
	mainNode := builder.Instance.Spec.MainNode

	statefulSpec.Spec.Template.Labels = metadata.GetLabels(mainNode.Name, builder.Instance.Labels)
	ConfigureStatefulSetWithDefaultSettings(statefulSpec)
	ConfigureStatefulSetWithNodeSettings(mainNode, statefulSpec)
	ConfigureStatefulSetWithGlobalSettings(builder.Instance, statefulSpec)

	var container v12.Container
	ConfigureContainerWithDefaultSettings(&container)
	ConfigureContainerWithNodeSettings(mainNode, &container)
	ConfigureContainerWithGlobalSettings(builder.Instance, &container)
	envVars := BuildEnvVariablesFromGlobalAndNodeSpecificSettings(builder.Instance, mainNode)
	container.Env = envVars

	statefulSpec.Spec.Template.Spec.Containers = []v12.Container{container}

	if err := controllerutil.SetControllerReference(builder.Instance, statefulSpec, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder *StatefulSetBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	return []client.Object{}, nil
}
