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

	//rework creation of shared resources by separating computed values into a separate struct inside Builder struct. Reuse them accordingly
	volumeMounts := volumeMountsBuilder(builder.Instance)

	var defaultValues = map[string]any{
		"nodeId":  builder.Instance.Name,
		"dataDir": volumeMounts[0].MountPath,
	}

	var volumes []v12.Volume

	secretVolume := databaseSecretVolumeBuilder(builder.Instance)
	if secretVolume.Name != "" {
		volumes = append(volumes, secretVolume)
	}

	var initContainers []v12.Container

	dirSetupContainer := initContainerSpecBuilder(volumeMounts, defaultValues)
	if dirSetupContainer.Name != "" {
		initContainers = append(initContainers, dirSetupContainer)
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
					Volumes:        volumes,
					InitContainers: initContainers,
					Containers:     []v12.Container{containerSpecBuilder(builder.Instance, volumeMounts, defaultValues)},
				},
			},
		},
	}, nil
}

func (builder *StatefulSetBuilder) Update(object client.Object) error {
	statefulSet := object.(*v1.StatefulSet)

	statefulSet.Spec.Replicas = builder.Instance.Spec.Replicas
	statefulSet.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)

	statefulSet.Spec.Template.Labels = metadata.Label(builder.Instance.Name)
	statefulSet.Spec.Template.Spec.SecurityContext = &builder.Instance.Spec.PodSecurityContext
	if statefulSet.Spec.Template.Spec.Containers == nil {
		statefulSet.Spec.Template.Spec.Containers = []v12.Container{containerSpecBuilder(builder.Instance)}
	} else {
		statefulSet.Spec.Template.Spec.Containers[0] = containerSpecBuilder(builder.Instance)
	}

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

func initContainerSpecBuilder(volumeMounts []v12.VolumeMount, defaultValues map[string]any) (container v12.Container) {
	container = v12.Container{
		Name:    DIR_SETUP_CONTAINER_NAME,
		Image:   DIR_SETUP_CONTAINER_IMAGE,
		Command: []string{"echo hello"},
	}
	container.VolumeMounts = volumeMounts

	return
}

func containerSpecBuilder(instance *v1alpha1.TeamCity, volumeMounts []v12.VolumeMount, defaultValues map[string]any) v12.Container {
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

	secretVolumeMounts := secretMountsBuilder(instance, defaultValues["dataDir"])
	volumeMounts = append(volumeMounts, secretVolumeMounts...)

	container.VolumeMounts = volumeMounts

	var envVarDefaults = map[string]string{
		"TEAMCITY_SERVER_MEM_OPTS": fmt.Sprintf("%s%s", "-Xmx", xmxValueCalculator(instance.Spec.XmxPercentage, container.Resources.Requests.Memory().Value())),
		"TEAMCITY_DATA_PATH":       fmt.Sprintf("%s", defaultValues["dataDir"]),
		"TEAMCITY_LOGS_PATH":       fmt.Sprintf("%s%s", defaultValues["dataDir"], "/logs"),
		"TEAMCITY_SERVER_OPTS": "-XX:+HeapDumpOnOutOfMemoryError -XX:+DisableExplicitGC" +
			fmt.Sprintf(" -XX:HeapDumpPath=%s%s%s", defaultValues["dataDir"], "/memoryDumps/", defaultValues["nodeId"]) +
			fmt.Sprintf(" -Dteamcity.server.nodeId=%s", defaultValues["nodeId"]) +
			fmt.Sprintf(" -Dteamcity.server.rootURL=%s", defaultValues["nodeId"]),
	}
	container.Env = environmentVariablesBuilder(instance, envVarDefaults)

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

func environmentVariablesBuilder(instance *v1alpha1.TeamCity, envVarDefaults map[string]string) (envVars []v12.EnvVar) {
	// merge with defaults
	envVars = []v12.EnvVar{}

	mergedMaps := make(map[string]string)
	for k, v := range envVarDefaults {
		mergedMaps[k] = v
	}
	for k, v := range instance.Spec.Env {
		mergedMaps[k] = v
	}

	for key, value := range mergedMaps {
		var envVar = v12.EnvVar{
			Name:      key,
			Value:     value,
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
func secretMountsBuilder(instance *v1alpha1.TeamCity, dataDirPath any) (volumeMounts []v12.VolumeMount) {
	if instance.Spec.DatabaseSecretName != "" {
		volumeMounts = append(volumeMounts, v12.VolumeMount{Name: DATABASE_PROPERTIES_VOLUME_NAME, MountPath: fmt.Sprintf("%s/config", dataDirPath)})
	}
	return
}

func databaseSecretVolumeBuilder(instance *v1alpha1.TeamCity) (volume v12.Volume) {
	if instance.Spec.DatabaseSecretName != "" {
		volume = v12.Volume{
			Name: DATABASE_PROPERTIES_VOLUME_NAME,
			VolumeSource: v12.VolumeSource{
				Secret: &v12.SecretVolumeSource{
					SecretName: instance.Spec.DatabaseSecretName,
				},
			},
		}
	}
	return
}
