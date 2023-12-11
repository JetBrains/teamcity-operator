package v1alpha1

import (
	"git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this TeamCity to the Hub version (v1beta1).
func (src *TeamCity) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.TeamCity)

	//version specific conversion
	dataDirIndex := 0
	dataDir := src.Spec.PersistentVolumeClaims[dataDirIndex]
	dst.Spec.DataDirVolumeClaim = v1beta1.CustomPersistentVolumeClaim(dataDir)
	//remove data dir from list of pvcs
	trimmedPersistentVolumeClaims := src.Spec.PersistentVolumeClaims[dataDirIndex+1:]
	//add other elements into a new type
	var newPersistentVolumeClaimList []v1beta1.CustomPersistentVolumeClaim
	for _, element := range trimmedPersistentVolumeClaims {
		newPersistentVolumeClaimList = append(newPersistentVolumeClaimList, v1beta1.CustomPersistentVolumeClaim(element))
	}
	//save list of persistent volumes into a new object
	dst.Spec.PersistentVolumeClaims = newPersistentVolumeClaimList
	//the rest fields that remain the same
	//object meta
	dst.ObjectMeta = src.ObjectMeta

	//object spec
	dst.Spec.Env = src.Spec.Env
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Requests = src.Spec.Requests
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Limits = src.Spec.Limits
	dst.Spec.TeamCityServerPort = src.Spec.TeamCityServerPort
	dst.Spec.PodSecurityContext = src.Spec.PodSecurityContext
	dst.Spec.XmxPercentage = src.Spec.XmxPercentage

	dst.Spec.HealthEndpoint = src.Spec.HealthEndpoint
	dst.Spec.ReadinessEndpoint = src.Spec.ReadinessEndpoint
	dst.Spec.ReadinessProbeSettings = src.Spec.ReadinessProbeSettings
	dst.Spec.LivenessProbeSettings = src.Spec.LivenessProbeSettings
	dst.Spec.StartupProbeSettings = src.Spec.StartupProbeSettings

	dst.Spec.InitContainers = src.Spec.InitContainers
	dst.Spec.StartupPropertiesConfig = src.Spec.StartupPropertiesConfig
	dst.Spec.DatabaseSecret = v1beta1.DatabaseSecret(src.Spec.DatabaseSecret)

	//object status
	dst.Status = v1beta1.TeamCityStatus(src.Status)
	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this version.
func (dst *TeamCity) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.TeamCity)

	//version specific conversion
	dataDirPersistentVolumeClaim := CustomPersistentVolumeClaim(src.Spec.DataDirVolumeClaim)
	var persistentVolumeClaimList []CustomPersistentVolumeClaim
	persistentVolumeClaimList = append(persistentVolumeClaimList, dataDirPersistentVolumeClaim)
	for _, elem := range src.Spec.PersistentVolumeClaims {
		persistentVolumeClaimList = append(persistentVolumeClaimList, CustomPersistentVolumeClaim(elem))
	}
	dst.Spec.PersistentVolumeClaims = persistentVolumeClaimList
	//object meta
	dst.ObjectMeta = src.ObjectMeta

	//object spec
	dst.Spec.Env = src.Spec.Env
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Requests = src.Spec.Requests
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Limits = src.Spec.Limits
	dst.Spec.TeamCityServerPort = src.Spec.TeamCityServerPort
	dst.Spec.PodSecurityContext = src.Spec.PodSecurityContext
	dst.Spec.XmxPercentage = src.Spec.XmxPercentage

	dst.Spec.HealthEndpoint = src.Spec.HealthEndpoint
	dst.Spec.ReadinessEndpoint = src.Spec.ReadinessEndpoint
	dst.Spec.ReadinessProbeSettings = src.Spec.ReadinessProbeSettings
	dst.Spec.LivenessProbeSettings = src.Spec.LivenessProbeSettings
	dst.Spec.StartupProbeSettings = src.Spec.StartupProbeSettings

	dst.Spec.InitContainers = src.Spec.InitContainers
	dst.Spec.StartupPropertiesConfig = src.Spec.StartupPropertiesConfig
	dst.Spec.DatabaseSecret = DatabaseSecret(src.Spec.DatabaseSecret)

	dst.Status = TeamCityStatus(src.Status)
	return nil
}
