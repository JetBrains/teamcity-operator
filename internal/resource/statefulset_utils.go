package resource

import (
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"strings"
)

const (
	DATABASE_PROPERTIES_VOLUME_NAME         = "database-properties"
	TEAMCITY_DATABASE_PROPERTIES_MOUNT_PATH = "/config/database.properties"
	TEAMCITY_DATABASE_PROPERTIES_SUB_PATH   = "database.properties"
	TEAMCITY_CONTAINER_NAME                 = "teamcity-server"
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
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
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

func XmxValueCalculator(percentage int64, requestedMemoryValue int64) (xmxValue string) {
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
func ConfigureContainer(instance *TeamCity, node Node, container *v12.Container) {
	container.Name = TEAMCITY_CONTAINER_NAME
	container.Image = instance.Spec.Image
	container.ImagePullPolicy = v12.PullIfNotPresent

	container.LivenessProbe = &node.Spec.LivenessProbeSettings
	container.ReadinessProbe = &node.Spec.ReadinessProbeSettings
	container.StartupProbe = &node.Spec.StartupProbeSettings
	container.Resources.Limits = node.Spec.Limits
	container.Resources.Requests = node.Spec.Limits

	container.Ports = []v12.ContainerPort{instance.Spec.TeamCityServerPort}
	container.LivenessProbe.ProbeHandler.HTTPGet = &instance.Spec.ReadinessEndpoint
	container.ReadinessProbe.ProbeHandler.HTTPGet = &instance.Spec.ReadinessEndpoint
	container.StartupProbe.ProbeHandler.HTTPGet = &instance.Spec.HealthEndpoint
	allPersistentVolumeClaims := instance.GetAllCustomPersistentVolumeClaim()
	volumeMounts := BuildVolumeMountsFromPersistentVolumeClaims(allPersistentVolumeClaims)
	container.VolumeMounts = volumeMounts
	envVars := BuildEnvVariablesFromGlobalAndNodeSpecificSettings(instance, node)
	container.Env = envVars

}

func ConfigureStatefulSet(instance *TeamCity, node Node, current *v1.StatefulSet) {
	allPersistentVolumeClaims := instance.GetAllCustomPersistentVolumeClaim()
	volumes := BuildVolumesFromPersistentVolumeClaims(allPersistentVolumeClaims)
	current.Spec.Replicas = pointer.Int32(1)
	current.Spec.Template.Annotations = node.Annotations
	current.Spec.Template.Spec.Volumes = volumes
	current.Spec.Template.Spec.InitContainers = node.Spec.InitContainers
	current.Spec.Template.Spec.NodeSelector = node.Spec.NodeSelector
	current.Spec.Template.Spec.Affinity = &node.Spec.Affinity
	current.Spec.Template.Spec.SecurityContext = &node.Spec.PodSecurityContext
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

func DatabaseEnvVarBuilder(databaseSecretName string) []v12.EnvVar {
	return []v12.EnvVar{
		{
			Name: "TEAMCITY_DB_USER",
			ValueFrom: &v12.EnvVarSource{
				SecretKeyRef: &v12.SecretKeySelector{
					LocalObjectReference: v12.LocalObjectReference{Name: databaseSecretName},
					Key:                  "connectionProperties.user",
				},
			},
		},
		{
			Name: "TEAMCITY_DB_PASSWORD",
			ValueFrom: &v12.EnvVarSource{
				SecretKeyRef: &v12.SecretKeySelector{
					LocalObjectReference: v12.LocalObjectReference{Name: databaseSecretName},
					Key:                  "connectionProperties.password",
				},
			},
		},
		{
			Name: "TEAMCITY_DB_URL",
			ValueFrom: &v12.EnvVarSource{
				SecretKeyRef: &v12.SecretKeySelector{
					LocalObjectReference: v12.LocalObjectReference{Name: databaseSecretName},
					Key:                  "connectionUrl",
				},
			},
		},
	}
}

func BuildEnvVariablesFromGlobalAndNodeSpecificSettings(instance *TeamCity, node Node) []v12.EnvVar {
	dataDirPath := instance.DataDirPath()
	extraServerOpts := ConvertStartUpPropertiesToServerOptions(instance.Spec.StartupPropertiesConfig)
	var responsibilities string
	if len(node.Spec.Responsibilities) > 0 {
		responsibilities = ConvertResponsibilitiesToServerOptions(node.Spec.Responsibilities)
	}
	extraServerOpts = extraServerOpts + responsibilities
	xmxValue := XmxValueCalculator(instance.Spec.XmxPercentage, node.Spec.Requests.Memory().Value())
	envVars := DefaultEnvironmentVariableBuilder(node.Name, xmxValue, dataDirPath, extraServerOpts)
	nodeSpecificEnvVars := ConvertNodeEnvVars(node.Spec.Env)
	envVars = append(envVars, nodeSpecificEnvVars...)
	if instance.DatabaseSecretProvided() {
		databaseEnvVars := DatabaseEnvVarBuilder(instance.Spec.DatabaseSecret.Secret)
		envVars = append(envVars, databaseEnvVars...)
	}
	return envVars
}

func ConvertResponsibilitiesToServerOptions(responsibilities []string) string {
	stringResponsibilities := strings.Join(responsibilities, ",")
	return fmt.Sprintf(" -Dteamcity.server.responsibilities=%s", stringResponsibilities)

}
