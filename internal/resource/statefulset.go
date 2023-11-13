package resource

import (
	"fmt"
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DATABASE_PROPERTIES_VOLUME_NAME         = "database-properties"
	TEAMCITY_DATABASE_PROPERTIES_MOUNT_PATH = "/config/database.properties"
	TEAMCITY_DATABASE_PROPERTIES_SUB_PATH   = "database.properties"
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

func (builder *StatefulSetBuilder) Build() (client.Object, error) {
	pvcList, err := persistentVolumeClaimTemplatesBuild(builder.Instance, builder.Scheme)

	if err != nil {
		return nil, err
	}

	return &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Instance.Name,
			Namespace: builder.Instance.Namespace,
		},
		Spec: v1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: metadata.LabelSelector(builder.Instance.Name),
			},

			VolumeClaimTemplates: pvcList,
			Template: v12.PodTemplateSpec{
				Spec: v12.PodSpec{
					InitContainers: []v12.Container{},
					Containers:     []v12.Container{},
				},
			},
		},
	}, nil
}

func (builder *StatefulSetBuilder) Update(object client.Object) error {
	var dataDirVolumeMount v12.VolumeMount
	var volumes []v12.Volume
	var extraServerOpts string
	var extraEnvVars = make(map[string]string)

	statefulSet := object.(*v1.StatefulSet)

	volumeMounts := volumeMountsBuilder(builder.Instance)

	dataDirVolumeMount, _ = builder.defineTeamCityDirectories(volumeMounts)
	dataDirPath := dataDirVolumeMount.MountPath

	initContainers := builder.Instance.Spec.InitContainers

	if builder.Instance.DatabaseSecretProvided() {
		secretVolume := databaseSecretVolumeBuilder(builder.Instance.Spec.DatabaseSecret.Secret)
		volumes = append(volumes, secretVolume)
		secretVolumeMounts := secretMountsBuilder(dataDirPath)
		volumeMounts = append(volumeMounts, secretVolumeMounts)
	}

	if builder.Instance.StartUpPropertiesConfigProvided() {
		extraServerOpts = builder.convertStartUpPropertiesToServerOptions()
	}

	defaultEnvVars := builder.defaultEnvironmentVariableBuilder(dataDirPath, extraServerOpts)

	envVars := builder.environmentVariablesBuilder(defaultEnvVars, extraEnvVars)

	statefulSet.Spec.Replicas = builder.Instance.Spec.Replicas
	statefulSet.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)

	statefulSet.Spec.Template.Labels = metadata.Label(builder.Instance.Name)
	statefulSet.Spec.Template.Spec.SecurityContext = &builder.Instance.Spec.PodSecurityContext

	statefulSet.Spec.Template.Spec.Volumes = volumes
	statefulSet.Spec.Template.Spec.InitContainers = initContainers

	teamcityContainer := builder.containerSpecBuilder(volumeMounts, envVars)
	statefulSet.Spec.Template.Spec.Containers = []v12.Container{teamcityContainer}

	if err := controllerutil.SetControllerReference(builder.Instance, statefulSet, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func persistentVolumeClaimTemplatesBuild(instance *v1alpha1.TeamCity, scheme *runtime.Scheme) ([]v12.PersistentVolumeClaim, error) {
	var pvcList []v12.PersistentVolumeClaim
	for _, claim := range instance.Spec.PersistentVolumeClaims {
		pvc := v12.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      claim.Name,
				Namespace: instance.Namespace,
			},
			Spec: claim.Spec,
		}
		if err := controllerutil.SetControllerReference(instance, &pvc, scheme); err != nil {
			return []v12.PersistentVolumeClaim{}, fmt.Errorf("failed setting controller reference: %w", err)
		}
		pvcList = append(pvcList, pvc)
	}
	return pvcList, nil
}

func (builder *StatefulSetBuilder) containerSpecBuilder(volumeMounts []v12.VolumeMount, env []v12.EnvVar) v12.Container {
	instance := builder.Instance
	var container = v12.Container{
		Name:  instance.Name,
		Image: instance.Spec.Image,
	}

	container.ImagePullPolicy = v12.PullIfNotPresent

	container.Ports = append([]v12.ContainerPort{}, instance.Spec.TeamCityServerPort)
	container.Lifecycle = lifecycleOptionsBuilder()

	container.LivenessProbe = &instance.Spec.LivenessProbeSettings
	container.ReadinessProbe = &instance.Spec.ReadinessProbeSettings
	container.StartupProbe = &instance.Spec.StartupProbeSettings

	container.LivenessProbe.ProbeHandler.HTTPGet = &instance.Spec.ReadinessEndpoint
	container.ReadinessProbe.ProbeHandler.HTTPGet = &instance.Spec.ReadinessEndpoint
	container.StartupProbe.ProbeHandler.HTTPGet = &instance.Spec.HealthEndpoint

	container.Resources.Limits = instance.Spec.Limits
	container.Resources.Requests = instance.Spec.Requests

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

func (builder *StatefulSetBuilder) defaultEnvironmentVariableBuilder(dataDirPath string, extraServerOpts string) map[string]string {
	return map[string]string{
		"TEAMCITY_SERVER_MEM_OPTS": fmt.Sprintf("%s%s", "-Xmx", xmxValueCalculator(builder.Instance.Spec.XmxPercentage, builder.Instance.Spec.Requests.Memory().Value())),
		"TEAMCITY_DATA_PATH":       fmt.Sprintf("%s", dataDirPath),
		"TEAMCITY_LOGS_PATH":       fmt.Sprintf("%s%s", dataDirPath, "/logs"),
		"TEAMCITY_SERVER_OPTS": "-XX:+HeapDumpOnOutOfMemoryError -XX:+DisableExplicitGC" +
			fmt.Sprintf(" -XX:HeapDumpPath=%s%s%s", dataDirPath, "/memoryDumps/", builder.Instance.Name) +
			fmt.Sprintf(" -Dteamcity.server.nodeId=%s", builder.Instance.Name) +
			fmt.Sprintf(" -Dteamcity.server.rootURL=%s", builder.Instance.Name) +
			extraServerOpts,
	}
}

func (builder *StatefulSetBuilder) environmentVariablesBuilder(envVarDefaults map[string]string, extraVars map[string]string) (envVars []v12.EnvVar) {

	// merge with defaults
	envVars = []v12.EnvVar{}

	mergedMaps := make(map[string]string)
	for k, v := range envVarDefaults {
		mergedMaps[k] = v
	}
	for k, v := range builder.Instance.Spec.Env {
		mergedMaps[k] = v
	}
	for k, v := range extraVars {
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

func volumeMountsBuilder(instance *v1alpha1.TeamCity) (volumeMounts []v12.VolumeMount) {
	for _, claim := range instance.Spec.PersistentVolumeClaims {
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

func (builder *StatefulSetBuilder) defineTeamCityDirectories(mounts []v12.VolumeMount) (dataDir v12.VolumeMount, configDir v12.VolumeMount) {
	if len(mounts) > 0 {
		dataDir = mounts[0]
	}
	if len(mounts) > 1 {
		configDir = mounts[1]
	} else {
		configDir = dataDir
	}
	return dataDir, configDir
}
