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

const (
	DATABASE_PROPERTIES_VOLUME_NAME         = "database-properties"
	TEAMCITY_DATABASE_PROPERTIES_MOUNT_PATH = "/config/database.properties"
	TEAMCITY_DATABASE_PROPERTIES_SUB_PATH   = "database.properties"
	TEAMCITY_CONTAINER_NAME                 = "teamcity-server"
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
	dataDirPath := builder.Instance.DataDirPath()

	//if builder.Instance.DatabaseSecretProvided() {
	//	secretVolume := DatabaseSecretVolumeBuilder(builder.Instance.Spec.DatabaseSecret.Secret)
	//	volumes = append(volumes, secretVolume)
	//	secretVolumeMounts := SecretMountsBuilder(dataDirPath)
	//	volumeMounts = append(volumeMounts, secretVolumeMounts)
	//}

	statefulSpec.Spec.Template.Labels = metadata.GetLabels(mainNode.Name, builder.Instance.Labels)
	ConfigureStatefulSetWithDefaultSettings(statefulSpec)
	ConfigureStatefulSetWithNodeSettings(mainNode, statefulSpec)
	ConfigureStatefulSetWithGlobalSettings(builder.Instance, statefulSpec)

	var container v12.Container
	ConfigureContainerWithDefaultSettings(&container)
	ConfigureContainerWithNodeSettings(mainNode, &container)
	ConfigureContainerWithGlobalSettings(builder.Instance, &container)

	extraServerOpts := ConvertStartUpPropertiesToServerOptions(builder.Instance.Spec.StartupPropertiesConfig)
	xmxValue := xmxValueCalculator(builder.Instance.Spec.XmxPercentage, mainNode.Requests.Memory().Value())
	defaultEnvVars := DefaultEnvironmentVariableBuilder(mainNode.Name, xmxValue, dataDirPath, extraServerOpts)
	nodeEnvVars := ConvertNodeEnvVars(mainNode.Env)
	envVars := append(defaultEnvVars, nodeEnvVars...)
	if builder.Instance.DatabaseSecretProvided() {
		databaseEnvVars := DatabaseEnvVarBuilder(builder.Instance.Spec.DatabaseSecret.Secret)
		envVars = append(envVars, databaseEnvVars...)
	}
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
