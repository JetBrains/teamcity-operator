package resource

import (
	"context"
	"errors"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
	var objectList []client.Object
	mainNodeLabels := metadata.GetLabels(builder.Instance.Spec.MainNode.Name, builder.Instance.Labels)
	mainNode := builder.BuildEmptyStatefulSet(builder.Instance.Spec.MainNode.Name, mainNodeLabels)
	objectList = append(objectList, &mainNode)
	for _, secondaryNode := range builder.Instance.Spec.SecondaryNodes {
		nodeLabels := metadata.GetLabels(secondaryNode.Name, builder.Instance.Labels)
		node := builder.BuildEmptyStatefulSet(secondaryNode.Name, nodeLabels)
		objectList = append(objectList, &node)
	}
	return objectList, nil
}

func (builder *StatefulSetBuilder) BuildEmptyStatefulSet(statefulSetName string, labels map[string]string) v1.StatefulSet {
	return v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetName,
			Namespace: builder.Instance.Namespace,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			Template: v12.PodTemplateSpec{
				Spec: v12.PodSpec{
					InitContainers: []v12.Container{},
					Containers:     []v12.Container{},
				},
			},
		},
	}
}

func (builder *StatefulSetBuilder) Update(object client.Object) error {
	var idx int
	var volumes []v12.Volume
	var extraServerOpts string
	nodeList := append(builder.Instance.Spec.SecondaryNodes, builder.Instance.Spec.MainNode)
	if idx = builder.getNodeIndex(object, nodeList); idx == -1 {
		return fmt.Errorf("failed to update object: %w", errors.New("the specified StatefulSet does not exist: "+object.GetName()))
	}

	desired := nodeList[idx]
	current := object.(*v1.StatefulSet)

	volumes = builder.volumeBuilders()

	volumeMounts := builder.volumeMountsBuilder()

	dataDirPath := builder.Instance.DataDirPath()

	initContainers := desired.InitContainers

	if builder.Instance.DatabaseSecretProvided() {
		secretVolume := databaseSecretVolumeBuilder(builder.Instance.Spec.DatabaseSecret.Secret)
		volumes = append(volumes, secretVolume)
		secretVolumeMounts := secretMountsBuilder(dataDirPath)
		volumeMounts = append(volumeMounts, secretVolumeMounts)
	}

	if builder.Instance.StartUpPropertiesConfigProvided() {
		extraServerOpts = builder.convertStartUpPropertiesToServerOptions()
	}

	defaultEnvVars := builder.defaultEnvironmentVariableBuilder(desired.Name, desired.Requests.Memory().Value(), dataDirPath, extraServerOpts)
	envVars := builder.environmentVariablesBuilder(defaultEnvVars, desired.Env)
	podIPEnvVar := builder.podIPEnvVariableBuilder()
	envVars = append([]v12.EnvVar{podIPEnvVar}, envVars...) //MY_IP var should be specified before options
	current.Spec.Replicas = pointer.Int32(1)
	current.Spec.Template.Labels = metadata.GetLabels(desired.Name, builder.Instance.Labels)
	current.Spec.Template.Spec.SecurityContext = &desired.PodSecurityContext

	current.Spec.Template.Spec.Volumes = volumes
	current.Spec.Template.Spec.InitContainers = initContainers

	teamcityContainer := builder.containerSpecBuilder(desired, volumeMounts, envVars)
	current.Spec.Template.Spec.Containers = []v12.Container{teamcityContainer}
	current.Spec.Template.Spec.NodeSelector = desired.NodeSelector
	current.Spec.Template.Spec.Affinity = &desired.Affinity

	if err := controllerutil.SetControllerReference(builder.Instance, current, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder *StatefulSetBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	return []client.Object{}, nil
}

func (builder *StatefulSetBuilder) containerSpecBuilder(node Node, volumeMounts []v12.VolumeMount, env []v12.EnvVar) v12.Container {
	var container = v12.Container{
		Name:  TEAMCITY_CONTAINER_NAME,
		Image: builder.Instance.Spec.Image,
	}

	container.ImagePullPolicy = v12.PullIfNotPresent

	container.Ports = append([]v12.ContainerPort{}, builder.Instance.Spec.TeamCityServerPort)
	container.Lifecycle = lifecycleOptionsBuilder()

	container.LivenessProbe = &node.LivenessProbeSettings
	container.ReadinessProbe = &node.ReadinessProbeSettings
	container.StartupProbe = &node.StartupProbeSettings

	container.LivenessProbe.ProbeHandler.HTTPGet = &builder.Instance.Spec.ReadinessEndpoint
	container.ReadinessProbe.ProbeHandler.HTTPGet = &builder.Instance.Spec.ReadinessEndpoint
	container.StartupProbe.ProbeHandler.HTTPGet = &builder.Instance.Spec.HealthEndpoint

	container.Resources.Limits = node.Limits
	container.Resources.Requests = node.Requests

	container.VolumeMounts = volumeMounts

	container.Env = env

	return container
}

func xmxValueCalculator(percentage int64, requestedMemoryValue int64) (xmxValue *resource.Quantity) {
	ratio := float64(percentage) / 100
	xmxValue = resource.NewQuantity(int64(ratio*float64(requestedMemoryValue)), resource.DecimalSI)
	return
}

func lifecycleOptionsBuilder() (lifecycle *v12.Lifecycle) {
	lifecycle = &v12.Lifecycle{
		PostStart: nil,
		PreStop: &v12.LifecycleHandler{
			Exec: &v12.ExecAction{
				Command: []string{"/bin/sh", "-c", "/opt/teamcity/bin/shutdown.sh"},
			},
			HTTPGet: nil,
		},
	}
	return
}

func (builder *StatefulSetBuilder) defaultEnvironmentVariableBuilder(nodeName string, memoryValue int64, dataDirPath string, extraServerOpts string) map[string]string {
	return map[string]string{
		"TEAMCITY_SERVER_MEM_OPTS": fmt.Sprintf("%s%s", "-Xmx", xmxValueCalculator(builder.Instance.Spec.XmxPercentage, memoryValue)),
		"TEAMCITY_DATA_PATH":       fmt.Sprintf("%s", dataDirPath),
		"TEAMCITY_LOGS_PATH":       fmt.Sprintf("%s%s", dataDirPath, "/logs"),
		"TEAMCITY_SERVER_OPTS": "-XX:+HeapDumpOnOutOfMemoryError -XX:+DisableExplicitGC" +
			fmt.Sprintf(" -XX:HeapDumpPath=%s%s%s", dataDirPath, "/memoryDumps/", nodeName) +
			fmt.Sprintf(" -Dteamcity.server.nodeId=%s", nodeName) +
			fmt.Sprintf(" -Dteamcity.server.rootURL=http://$(MY_IP)") +
			extraServerOpts,
	}
}

func (builder *StatefulSetBuilder) environmentVariablesBuilder(envVarDefaults map[string]string, nodeEnvVars map[string]string) (envVars []v12.EnvVar) {

	// merge with defaults
	envVars = []v12.EnvVar{}

	mergedMaps := make(map[string]string)
	for k, v := range envVarDefaults {
		mergedMaps[k] = v
	}
	for k, v := range nodeEnvVars {
		mergedMaps[k] = v
	}

	//if we don't sort keys we might produce a different env variable array each time
	keys := SortKeysAlphabeticallyInMap(mergedMaps)

	for _, k := range keys {
		var envVar = v12.EnvVar{
			Name:      k,
			Value:     mergedMaps[k],
			ValueFrom: nil,
		}
		envVars = append(envVars, envVar)
	}
	return
}

func (builder *StatefulSetBuilder) podIPEnvVariableBuilder() v12.EnvVar {
	return v12.EnvVar{
		Name: "MY_IP",
		ValueFrom: &v12.EnvVarSource{
			FieldRef: &v12.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	}
}

func (builder *StatefulSetBuilder) volumeMountsBuilder() (volumeMounts []v12.VolumeMount) {
	volumeMounts = append(volumeMounts, v12.VolumeMount{Name: builder.Instance.Spec.DataDirVolumeClaim.VolumeMount.Name, MountPath: builder.Instance.Spec.DataDirVolumeClaim.VolumeMount.MountPath})
	for _, claim := range builder.Instance.Spec.PersistentVolumeClaims {
		volumeMounts = append(volumeMounts, v12.VolumeMount{Name: claim.VolumeMount.Name, MountPath: claim.VolumeMount.MountPath})
	}
	return
}
func secretMountsBuilder(dataDirPath string) v12.VolumeMount {
	return v12.VolumeMount{Name: DATABASE_PROPERTIES_VOLUME_NAME, MountPath: fmt.Sprintf("%s%s", dataDirPath, TEAMCITY_DATABASE_PROPERTIES_MOUNT_PATH), SubPath: TEAMCITY_DATABASE_PROPERTIES_SUB_PATH}
}

func databaseSecretVolumeBuilder(databaseSecretName string) v12.Volume {
	return v12.Volume{
		Name: DATABASE_PROPERTIES_VOLUME_NAME,
		VolumeSource: v12.VolumeSource{
			Secret: &v12.SecretVolumeSource{
				SecretName: databaseSecretName,
			},
		},
	}
}

func (builder *StatefulSetBuilder) convertStartUpPropertiesToServerOptions() (res string) {
	sortedKeys := SortKeysAlphabeticallyInMap(builder.Instance.Spec.StartupPropertiesConfig)
	for _, k := range sortedKeys {
		res += fmt.Sprintf(" -D%s=%s", k, builder.Instance.Spec.StartupPropertiesConfig[k])
	}
	return
}

func (builder *StatefulSetBuilder) volumeBuilders() (volumes []v12.Volume) {
	volumes = append(volumes, v12.Volume{Name: builder.Instance.Spec.DataDirVolumeClaim.Name,
		VolumeSource: v12.VolumeSource{
			PersistentVolumeClaim: &v12.PersistentVolumeClaimVolumeSource{
				ClaimName: builder.Instance.Spec.DataDirVolumeClaim.Name},
		},
	},
	)
	for _, claim := range builder.Instance.Spec.PersistentVolumeClaims {
		volumes = append(volumes, v12.Volume{Name: claim.Name,
			VolumeSource: v12.VolumeSource{
				PersistentVolumeClaim: &v12.PersistentVolumeClaimVolumeSource{
					ClaimName: claim.Name},
			},
		},
		)
	}
	return
}

func (builder *StatefulSetBuilder) getNodeIndex(object client.Object, nodeList []Node) int {
	for idx, ingress := range nodeList {
		if ingress.Name == object.GetName() {
			return idx
		}
	}
	return -1
}
