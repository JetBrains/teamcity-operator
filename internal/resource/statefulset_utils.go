package resource

import (
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func CreateEmptyStatefulSet(name string, namespace string, labels map[string]string) v1.StatefulSet {
	return v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
func DefaultEnvironmentVariableBuilder(nodeName string, xmxValue string, dataDirPath string, extraServerOpts string) []v12.EnvVar {
	return []v12.EnvVar{
		PodIPEnvVariableBuilder(),
		DataDirPathEnvVar(dataDirPath),
		LogDirPathEnvVar(dataDirPath),
		ServerMemOptsEnvVar(xmxValue),
		ServerOptsEnvVar(dataDirPath, nodeName, extraServerOpts),
	}
}

func DataDirPathEnvVar(dataDirPath string) v12.EnvVar {
	return v12.EnvVar{
		Name:  "TEAMCITY_DATA_PATH",
		Value: fmt.Sprintf("%s", dataDirPath),
	}
}

func LogDirPathEnvVar(dataDirPath string) v12.EnvVar {
	return v12.EnvVar{
		Name:  "TEAMCITY_LOGS_PATH",
		Value: fmt.Sprintf("%s%s", dataDirPath, "/logs"),
	}
}

func ServerMemOptsEnvVar(xmxValue string) v12.EnvVar {
	return v12.EnvVar{
		Name:  "TEAMCITY_SERVER_MEM_OPTS",
		Value: fmt.Sprintf("%s%s", "-Xmx", xmxValue),
	}
}

func ServerOptsEnvVar(dataDirPath string, nodeName string, extraServerOpts string) v12.EnvVar {
	return v12.EnvVar{
		Name: "TEAMCITY_SERVER_OPTS",
		Value: "-XX:+HeapDumpOnOutOfMemoryError -XX:+DisableExplicitGC" +
			fmt.Sprintf(" -XX:HeapDumpPath=%s%s%s", dataDirPath, "/memoryDumps/", nodeName) +
			fmt.Sprintf(" -Dteamcity.server.nodeId=%s", nodeName) +
			fmt.Sprintf(" -Dteamcity.server.rootURL=http://$(MY_IP)") +
			extraServerOpts,
	}
}

func xmxValueCalculator(percentage int64, requestedMemoryValue int64) (xmxValue string) {
	ratio := float64(percentage) / 100
	xmxValue = resource.NewQuantity(int64(ratio*float64(requestedMemoryValue)), resource.DecimalSI).String()
	return
}

func LifecycleOptionsBuilder() (lifecycle *v12.Lifecycle) {
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

func EnvironmentVariablesBuilder(envVarDefaults map[string]string, nodeEnvVars map[string]string) (envVars []v12.EnvVar) {

	// merge with defaults
	envVars = []v12.EnvVar{}
	podIPEnvVar := PodIPEnvVariableBuilder()

	envVars = append(envVars, podIPEnvVar)
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

func DatabaseSecretVolumeBuilder(databaseSecretName string) v12.Volume {
	return v12.Volume{
		Name: DATABASE_PROPERTIES_VOLUME_NAME,
		VolumeSource: v12.VolumeSource{
			Secret: &v12.SecretVolumeSource{
				SecretName: databaseSecretName,
			},
		},
	}
}

func ConvertStartUpPropertiesToServerOptions(startupProperties map[string]string) (res string) {
	sortedKeys := SortKeysAlphabeticallyInMap(startupProperties)
	for _, k := range sortedKeys {
		res += fmt.Sprintf(" -D%s=%s", k, startupProperties[k])
	}
	return
}

func PodIPEnvVariableBuilder() v12.EnvVar {
	return v12.EnvVar{
		Name: "MY_IP",
		ValueFrom: &v12.EnvVarSource{
			FieldRef: &v12.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	}
}

func BuildVolumesFromPersistentVolumeClaims(persistentVolumeClaims []CustomPersistentVolumeClaim) (volumes []v12.Volume) {
	for _, claim := range persistentVolumeClaims {
		volumes = append(volumes, createVolumeFromCustomPersistentVolumeClaim(claim))
	}
	return
}

func createVolumeFromCustomPersistentVolumeClaim(persistentVolumeClaim CustomPersistentVolumeClaim) v12.Volume {
	return v12.Volume{Name: persistentVolumeClaim.Name,
		VolumeSource: v12.VolumeSource{
			PersistentVolumeClaim: &v12.PersistentVolumeClaimVolumeSource{
				ClaimName: persistentVolumeClaim.Name},
		},
	}

}

func BuildVolumeMountsFromPersistentVolumeClaims(persistentVolumeClaims []CustomPersistentVolumeClaim) (volumeMounts []v12.VolumeMount) {
	for _, claim := range persistentVolumeClaims {
		volumeMounts = append(volumeMounts, createVolumeMountFromCustomPersistentVolumeClaim(claim))
	}
	return
}

func createVolumeMountFromCustomPersistentVolumeClaim(persistentVolumeClaim CustomPersistentVolumeClaim) v12.VolumeMount {
	return v12.VolumeMount{Name: persistentVolumeClaim.VolumeMount.Name, MountPath: persistentVolumeClaim.VolumeMount.MountPath}
}

func SecretMountsBuilder(dataDirPath string) v12.VolumeMount {
	return v12.VolumeMount{Name: DATABASE_PROPERTIES_VOLUME_NAME, MountPath: fmt.Sprintf("%s%s", dataDirPath, TEAMCITY_DATABASE_PROPERTIES_MOUNT_PATH), SubPath: TEAMCITY_DATABASE_PROPERTIES_SUB_PATH}
}
func ConfigureContainerWithGlobalSettings(instance *TeamCity, container *v12.Container) {
	container.Name = TEAMCITY_CONTAINER_NAME
	container.Image = instance.Spec.Image
	container.ImagePullPolicy = v12.PullIfNotPresent
	container.ImagePullPolicy = v12.PullIfNotPresent

	container.Ports = []v12.ContainerPort{instance.Spec.TeamCityServerPort}
	container.LivenessProbe.ProbeHandler.HTTPGet = &instance.Spec.ReadinessEndpoint
	container.ReadinessProbe.ProbeHandler.HTTPGet = &instance.Spec.ReadinessEndpoint
	container.StartupProbe.ProbeHandler.HTTPGet = &instance.Spec.HealthEndpoint
	allPersistentVolumeClaims := instance.GetAllCustomPersistentVolumeClaim()
	volumeMounts := BuildVolumeMountsFromPersistentVolumeClaims(allPersistentVolumeClaims)
	container.VolumeMounts = volumeMounts

}

func ConfigureContainerWithNodeSettings(node Node, container *v12.Container) {
	container.ImagePullPolicy = v12.PullIfNotPresent
	container.LivenessProbe = &node.LivenessProbeSettings
	container.ReadinessProbe = &node.ReadinessProbeSettings
	container.StartupProbe = &node.StartupProbeSettings
	container.Resources.Limits = node.Limits
	container.Resources.Requests = node.Limits
}

func ConfigureStatefulSetWithNodeSettings(node Node, current *v1.StatefulSet) {
	current.Spec.Template.Spec.InitContainers = node.InitContainers
	current.Spec.Template.Spec.NodeSelector = node.NodeSelector
	current.Spec.Template.Spec.Affinity = &node.Affinity
	current.Spec.Template.Spec.SecurityContext = &node.PodSecurityContext
}

func ConfigureStatefulSetWithGlobalSettings(instance *TeamCity, current *v1.StatefulSet) {
	allPersistentVolumeClaims := instance.GetAllCustomPersistentVolumeClaim()
	volumes := BuildVolumesFromPersistentVolumeClaims(allPersistentVolumeClaims)
	current.Spec.Template.Spec.Volumes = volumes
}

func ConfigureContainerWithDefaultSettings(container *v12.Container) {
	container.Lifecycle = LifecycleOptionsBuilder()
}

func ConfigureStatefulSetWithDefaultSettings(current *v1.StatefulSet) {
	current.Spec.Replicas = pointer.Int32(1)
}

func ConvertNodeEnvVars(env map[string]string) (envVars []v12.EnvVar) {
	for _, k := range env {
		var envVar = v12.EnvVar{
			Name:      k,
			Value:     env[k],
			ValueFrom: nil,
		}
		envVars = append(envVars, envVar)
	}
	return envVars
}
