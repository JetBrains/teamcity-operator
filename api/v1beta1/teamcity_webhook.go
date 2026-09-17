/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
	"strings"
)

var allTeamCityResponsibilities = []string{
	"MAIN_NODE",
	"CAN_PROCESS_BUILD_MESSAGES",
	"CAN_CHECK_FOR_CHANGES",
	"CAN_PROCESS_BUILD_TRIGGERS",
	"CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS",
}

var minimumRequiredMainNodeResponsibilities = []string{
	"MAIN_NODE",
	"CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS",
}
var validSecondaryNodeResponsibilities = []string{
	"CAN_PROCESS_BUILD_MESSAGES",
	"CAN_CHECK_FOR_CHANGES",
	"CAN_PROCESS_BUILD_TRIGGERS",
	"CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS",
}
var validMainNodeResponsibilities = allTeamCityResponsibilities

// log is for logging in this package.
var teamcitylog = logf.Log.WithName("teamcity-resource")

func (instance *TeamCity) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(instance).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-jetbrains-com-v1beta1-teamcity,mutating=true,failurePolicy=fail,sideEffects=None,groups=jetbrains.com,resources=teamcities,verbs=create;update,versions=v1beta1,name=mv1beta1teamcity.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &TeamCity{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (instance *TeamCity) Default() {
	teamcitylog.Info("default", "name", instance.Name)

}

//+kubebuilder:webhook:path=/validate-jetbrains-com-v1beta1-teamcity,mutating=false,failurePolicy=fail,sideEffects=None,groups=jetbrains.com,resources=teamcities,verbs=create;update,versions=v1beta1,name=vv1beta1teamcity.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &TeamCity{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (instance *TeamCity) ValidateCreate() (admission.Warnings, error) {
	teamcitylog.Info("validate create", "name", instance.Name)
	return validateCommonFields(instance)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (instance *TeamCity) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	teamcitylog.Info("validate update", "name", instance.Name)
	oldTeamCity, ok := old.(*TeamCity)
	if !ok {
		return nil, fmt.Errorf("expected a TeamCity object but got %T", old)
	}

	warn, err := validateCommonFields(instance)
	if err != nil {
		return nil, err
	}

	if ServiceNameChangedInSpec(oldTeamCity, instance) {
		if !instance.AllowsStatefulSetRecreate() {
			return nil, fmt.Errorf(
				"changing spec.*.serviceName on an existing TeamCity requires annotation %s=%q. "+
					"Kubernetes does not allow StatefulSet spec.serviceName to be updated in place; "+
					"with the annotation, the operator will delete and recreate the affected StatefulSet(s) and restart the node(s)",
				AllowStsRecreateAnnotationKey,
				AllowStsRecreateAnnotationValue,
			)
		}
		warn = append(warn, admission.Warnings{
			"spec.*.serviceName changed: the affected StatefulSet(s) will be recreated and the node(s) restarted",
		}...)
	}

	return warn, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (instance *TeamCity) ValidateDelete() (admission.Warnings, error) {
	teamcitylog.Info("validate delete", "name", instance.Name)

	return nil, nil
}

func hostPathVolumeWarnings(teamcity *TeamCity) admission.Warnings {
	var warnings admission.Warnings
	appendWarnings := func(basePath string, volumes []TeamCityVolume) {
		for idx, volume := range volumes {
			if volume.HostPath == nil {
				continue
			}
			warnings = append(warnings, fmt.Sprintf(
				"%s[%d] (%q) uses a hostPath volume: its contents depend on the Kubernetes node the pod is scheduled to and are lost on rescheduling",
				basePath, idx, volume.Name))
		}
	}

	appendWarnings("teamcity.spec.volumes", teamcity.Spec.Volumes)
	nodes := append([]Node{teamcity.Spec.MainNode}, teamcity.Spec.SecondaryNodes...)
	paths := nodeObjectPaths(teamcity)
	for nodeIdx, node := range nodes {
		appendWarnings(paths[nodeIdx]+".spec.volumes", node.Spec.Volumes)
	}
	return warnings
}

func validateCommonFields(teamcity *TeamCity) (admission.Warnings, error) {
	if err := validateRequestsOfAllNodes(teamcity); err != nil {
		return nil, err
	}
	if err := validateXmxPercentage(teamcity); err != nil {
		return nil, err
	}
	if err := validateAllCustomPersistentVolumeClaimsInObject(teamcity); err != nil {
		return nil, err
	}
	warnings := hostPathVolumeWarnings(teamcity)
	if responsibilityWarning, err := validateResponsibilitiesOfAllNodes(teamcity); err != nil || responsibilityWarning != "" {
		return append(warnings, responsibilityWarning), err
	}
	return warnings, nil
}

func validateXmxPercentage(teamcity *TeamCity) (err error) {
	if teamcity.Spec.XmxPercentage <= 0 {
		return typed.ValidationError{
			Path:         "teamcity.spec.xmxPercentage",
			ErrorMessage: "Xmx percentage cannot be set to 0 or lower",
		}
	}
	return nil
}

func validateRequestsInNode(objectPath string, node Node) (err error) {
	if len(node.Spec.Requests.Memory().String()) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "requests.memory"),
			ErrorMessage: "Requested memory cannot be empty",
		}
	}
	return nil
}

func validateAllCustomPersistentVolumeClaimsInObject(teamcity *TeamCity) (err error) {
	if err = validateCustomPersistentVolumeClaim("teamcity.spec.dataDirVolumeClaim", teamcity.Spec.DataDirVolumeClaim); err != nil {
		return err
	}
	for idx, additionalVolumeClaim := range teamcity.Spec.PersistentVolumeClaims {
		if err = validateCustomPersistentVolumeClaim(fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d]", idx), additionalVolumeClaim); err != nil {
			return err
		}
	}
	if err = validateNodeDataDirVolumeClaim(teamcity); err != nil {
		return err
	}
	if err = validateNodePersistentVolumeClaims(teamcity); err != nil {
		return err
	}
	if err = validateVolumes(teamcity); err != nil {
		return err
	}
	return nil
}

func nodeObjectPaths(teamcity *TeamCity) []string {
	paths := []string{"teamcity.spec.mainNode"}
	for idx := range teamcity.Spec.SecondaryNodes {
		paths = append(paths, fmt.Sprintf("teamcity.spec.secondaryNodes[%d]", idx))
	}
	return paths
}

func sharedVolumeNames(teamcity *TeamCity) map[string]string {
	names := map[string]string{
		teamcity.Spec.DataDirVolumeClaim.VolumeMount.Name: "teamcity.spec.dataDirVolumeClaim.volumeMount.name",
	}
	for idx, additional := range teamcity.Spec.PersistentVolumeClaims {
		names[additional.VolumeMount.Name] = fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d].volumeMount.name", idx)
	}
	return names
}

func sharedMountPaths(teamcity *TeamCity) map[string]string {
	paths := map[string]string{
		teamcity.Spec.DataDirVolumeClaim.VolumeMount.MountPath: "teamcity.spec.dataDirVolumeClaim.volumeMount.mountPath",
	}
	for idx, additional := range teamcity.Spec.PersistentVolumeClaims {
		paths[additional.VolumeMount.MountPath] = fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d].volumeMount.mountPath", idx)
	}
	return paths
}

func sharedClaimNames(teamcity *TeamCity) map[string]string {
	names := map[string]string{
		teamcity.Spec.DataDirVolumeClaim.Name: "teamcity.spec.dataDirVolumeClaim.name",
	}
	for idx, additional := range teamcity.Spec.PersistentVolumeClaims {
		names[additional.Name] = fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d].name", idx)
	}
	return names
}

func validateNodePersistentVolumeClaims(teamcity *TeamCity) error {
	reservedClaimNames := sharedClaimNames(teamcity)
	claimNamesSeen := map[string]string{}
	for _, node := range teamcity.GetAllNodes() {
		if node.Spec.NodeDataDirVolumeClaim != nil {
			claimNamesSeen[node.Spec.NodeDataDirVolumeClaim.Name] = node.Name
		}
	}

	nodes := append([]Node{teamcity.Spec.MainNode}, teamcity.Spec.SecondaryNodes...)
	paths := nodeObjectPaths(teamcity)
	for nodeIdx, node := range nodes {
		reservedVolumeNames := sharedVolumeNames(teamcity)
		reservedMountPaths := sharedMountPaths(teamcity)
		if node.Spec.NodeDataDirVolumeClaim != nil {
			reservedVolumeNames[node.Spec.NodeDataDirVolumeClaim.VolumeMount.Name] = paths[nodeIdx] + ".spec.nodeDataDirVolumeClaim.volumeMount.name"
			reservedMountPaths[node.Spec.NodeDataDirVolumeClaim.VolumeMount.MountPath] = paths[nodeIdx] + ".spec.nodeDataDirVolumeClaim.volumeMount.mountPath"
		}

		for idx, claim := range node.Spec.PersistentVolumeClaims {
			claimPath := fmt.Sprintf("%s.spec.persistentVolumeClaims[%d]", paths[nodeIdx], idx)
			if claim.ExistingClaim {
				if err := validateNodeDataDirExistingClaim(claimPath, claim); err != nil {
					return err
				}
			} else {
				if err := validateCustomPersistentVolumeClaim(claimPath, claim); err != nil {
					return err
				}
			}
			if len(claim.Name) > maxKubernetesDNSSubdomainLength {
				return typed.ValidationError{
					Path:         claimPath + ".name",
					ErrorMessage: fmt.Sprintf("claim name %q exceeds %d characters", claim.Name, maxKubernetesDNSSubdomainLength),
				}
			}
			if path, exists := reservedClaimNames[claim.Name]; exists {
				return typed.ValidationError{
					Path:         claimPath + ".name",
					ErrorMessage: fmt.Sprintf("collides with %s", path),
				}
			}
			if previous, exists := claimNamesSeen[claim.Name]; exists {
				return typed.ValidationError{
					Path:         claimPath + ".name",
					ErrorMessage: fmt.Sprintf("nodes %q and %q resolve to the same PVC %q", previous, node.Name, claim.Name),
				}
			}
			claimNamesSeen[claim.Name] = node.Name

			if path, exists := reservedVolumeNames[claim.VolumeMount.Name]; exists {
				return typed.ValidationError{
					Path:         claimPath + ".volumeMount.name",
					ErrorMessage: fmt.Sprintf("collides with %s", path),
				}
			}
			reservedVolumeNames[claim.VolumeMount.Name] = claimPath + ".volumeMount.name"

			if path, exists := reservedMountPaths[claim.VolumeMount.MountPath]; exists {
				return typed.ValidationError{
					Path:         claimPath + ".volumeMount.mountPath",
					ErrorMessage: fmt.Sprintf("collides with %s", path),
				}
			}
			reservedMountPaths[claim.VolumeMount.MountPath] = claimPath + ".volumeMount.mountPath"
		}
	}
	return nil
}

func validateVolumeEntry(volumePath string, volume TeamCityVolume) error {
	if len(volume.Name) <= 0 {
		return typed.ValidationError{
			Path:         volumePath + ".name",
			ErrorMessage: "Volume name is not set",
		}
	}
	if volume.Mount != nil && len(volume.Mount.MountPath) <= 0 {
		return typed.ValidationError{
			Path:         volumePath + ".mount.mountPath",
			ErrorMessage: "Volume mount path is not set",
		}
	}
	return nil
}

func validateVolumes(teamcity *TeamCity) error {
	specVolumeNames := map[string]string{}
	for idx, volume := range teamcity.Spec.Volumes {
		volumePath := fmt.Sprintf("teamcity.spec.volumes[%d]", idx)
		if err := validateVolumeEntry(volumePath, volume); err != nil {
			return err
		}
		if previous, exists := specVolumeNames[volume.Name]; exists {
			return typed.ValidationError{
				Path:         volumePath + ".name",
				ErrorMessage: fmt.Sprintf("collides with %s", previous),
			}
		}
		specVolumeNames[volume.Name] = volumePath + ".name"
	}

	nodes := append([]Node{teamcity.Spec.MainNode}, teamcity.Spec.SecondaryNodes...)
	paths := nodeObjectPaths(teamcity)
	for nodeIdx, node := range nodes {
		reservedVolumeNames := sharedVolumeNames(teamcity)
		reservedMountPaths := sharedMountPaths(teamcity)
		for _, claim := range teamcity.NodePersistentVolumeClaimsFor(node) {
			reservedVolumeNames[claim.VolumeMount.Name] = fmt.Sprintf("a per-node claim on %q", node.Name)
			reservedMountPaths[claim.VolumeMount.MountPath] = fmt.Sprintf("a per-node claim on %q", node.Name)
		}

		specVolumeCount := len(teamcity.Spec.Volumes)
		for idx, volume := range teamcity.VolumesFor(node) {
			volumePath := fmt.Sprintf("teamcity.spec.volumes[%d]", idx)
			if idx >= specVolumeCount {
				volumePath = fmt.Sprintf("%s.spec.volumes[%d]", paths[nodeIdx], idx-specVolumeCount)
				if err := validateVolumeEntry(volumePath, volume); err != nil {
					return err
				}
			}
			if path, exists := reservedVolumeNames[volume.Name]; exists {
				return typed.ValidationError{
					Path:         volumePath + ".name",
					ErrorMessage: fmt.Sprintf("collides with %s on node %q", path, node.Name),
				}
			}
			reservedVolumeNames[volume.Name] = volumePath + ".name"

			if volume.Mount == nil {
				continue
			}
			if path, exists := reservedMountPaths[volume.Mount.MountPath]; exists {
				return typed.ValidationError{
					Path:         volumePath + ".mount.mountPath",
					ErrorMessage: fmt.Sprintf("collides with %s on node %q", path, node.Name),
				}
			}
			reservedMountPaths[volume.Mount.MountPath] = volumePath + ".mount.mountPath"
		}
	}

	return nil
}

func validateNodeDataDirVolumeClaim(teamcity *TeamCity) error {
	type nodeClaim struct {
		node       Node
		objectPath string
		claim      *CustomPersistentVolumeClaim
	}

	var configured []nodeClaim
	for idx, node := range append([]Node{teamcity.Spec.MainNode}, teamcity.Spec.SecondaryNodes...) {
		objectPath := "teamcity.spec.mainNode"
		if idx > 0 {
			objectPath = fmt.Sprintf("teamcity.spec.secondaryNodes[%d]", idx-1)
		}
		if node.Spec.NodeDataDirVolumeClaim == nil {
			continue
		}
		configured = append(configured, nodeClaim{
			node:       node,
			objectPath: objectPath,
			claim:      node.Spec.NodeDataDirVolumeClaim,
		})
	}
	if len(configured) == 0 {
		return nil
	}

	if _, exists := teamcity.Spec.StartupPropertiesConfig[resourceTeamCityNodeDataPathProperty]; exists {
		return typed.ValidationError{
			Path:         "teamcity.spec.startupPropertiesConfig",
			ErrorMessage: "teamcity.node.data.path is set by nodeDataDirVolumeClaim; remove it from startupPropertiesConfig",
		}
	}

	reservedClaimNames := map[string]string{
		teamcity.Spec.DataDirVolumeClaim.Name: "teamcity.spec.dataDirVolumeClaim.name",
	}
	for idx, additional := range teamcity.Spec.PersistentVolumeClaims {
		reservedClaimNames[additional.Name] = fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d].name", idx)
	}

	reservedVolumeNames := map[string]string{
		teamcity.Spec.DataDirVolumeClaim.VolumeMount.Name: "teamcity.spec.dataDirVolumeClaim.volumeMount.name",
	}
	reservedMountPaths := map[string]string{
		teamcity.Spec.DataDirVolumeClaim.VolumeMount.MountPath: "teamcity.spec.dataDirVolumeClaim.volumeMount.mountPath",
	}
	for idx, additional := range teamcity.Spec.PersistentVolumeClaims {
		reservedVolumeNames[additional.VolumeMount.Name] = fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d].volumeMount.name", idx)
		reservedMountPaths[additional.VolumeMount.MountPath] = fmt.Sprintf("teamcity.spec.persistentVolumeClaims[%d].volumeMount.mountPath", idx)
	}

	claimNamesSeen := map[string]string{}
	sharedVolumeName := configured[0].claim.VolumeMount.Name
	sharedMountPath := configured[0].claim.VolumeMount.MountPath

	for _, item := range configured {
		claimPath := fmt.Sprintf("%s.spec.nodeDataDirVolumeClaim", item.objectPath)
		if item.claim.ExistingClaim {
			if err := validateNodeDataDirExistingClaim(claimPath, *item.claim); err != nil {
				return err
			}
		} else {
			if err := validateCustomPersistentVolumeClaim(claimPath, *item.claim); err != nil {
				return err
			}
		}

		effectiveName := item.claim.Name
		if len(effectiveName) > maxKubernetesDNSSubdomainLength {
			return typed.ValidationError{
				Path:         claimPath + ".name",
				ErrorMessage: fmt.Sprintf("claim name %q exceeds %d characters", effectiveName, maxKubernetesDNSSubdomainLength),
			}
		}
		if path, exists := reservedClaimNames[effectiveName]; exists {
			return typed.ValidationError{
				Path:         claimPath + ".name",
				ErrorMessage: fmt.Sprintf("collides with %s", path),
			}
		}
		if previous, exists := claimNamesSeen[effectiveName]; exists {
			return typed.ValidationError{
				Path:         claimPath + ".name",
				ErrorMessage: fmt.Sprintf("nodes %q and %q resolve to the same node data PVC %q", previous, item.node.Name, effectiveName),
			}
		}
		claimNamesSeen[effectiveName] = item.node.Name

		if item.claim.VolumeMount.Name != sharedVolumeName {
			return typed.ValidationError{
				Path:         claimPath + ".volumeMount.name",
				ErrorMessage: fmt.Sprintf("must match other nodes (%q)", sharedVolumeName),
			}
		}
		if item.claim.VolumeMount.MountPath != sharedMountPath {
			return typed.ValidationError{
				Path:         claimPath + ".volumeMount.mountPath",
				ErrorMessage: fmt.Sprintf("must match other nodes (%q)", sharedMountPath),
			}
		}
	}

	if path, exists := reservedVolumeNames[sharedVolumeName]; exists {
		return typed.ValidationError{
			Path:         configured[0].objectPath + ".spec.nodeDataDirVolumeClaim.volumeMount.name",
			ErrorMessage: fmt.Sprintf("collides with %s", path),
		}
	}
	if path, exists := reservedMountPaths[sharedMountPath]; exists {
		return typed.ValidationError{
			Path:         configured[0].objectPath + ".spec.nodeDataDirVolumeClaim.volumeMount.mountPath",
			ErrorMessage: fmt.Sprintf("collides with %s", path),
		}
	}

	return nil
}

func validateNodeDataDirExistingClaim(objectPath string, claim CustomPersistentVolumeClaim) error {
	if len(claim.Name) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "name"),
			ErrorMessage: "Claim name is not set",
		}
	}
	if len(claim.VolumeMount.Name) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "volumeMount.name"),
			ErrorMessage: "Volume mount name is not set",
		}
	}
	if len(claim.VolumeMount.MountPath) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "volumeMount.mountPath"),
			ErrorMessage: "Volume mount path is not set",
		}
	}
	return nil
}

const maxKubernetesDNSSubdomainLength = 253
const resourceTeamCityNodeDataPathProperty = "teamcity.node.data.path"

func validateCustomPersistentVolumeClaim(objectPath string, claim CustomPersistentVolumeClaim) error {
	if len(claim.Name) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "name"),
			ErrorMessage: "Claim name is not set",
		}
	}
	if len(claim.VolumeMount.Name) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "volumeMount.name"),
			ErrorMessage: "Volume mount name is not set",
		}
	}
	if len(claim.VolumeMount.MountPath) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "volumeMount.mountPath"),
			ErrorMessage: "Volume mount path is not set",
		}
	}

	if len(claim.Spec.Resources.Requests.Storage().String()) <= 0 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "spec.resources.requests.storage"),
			ErrorMessage: "Storage request is not set",
		}
	}

	return nil
}
func validateRequestsOfAllNodes(teamcity *TeamCity) (err error) {
	if err := validateRequestsInNode("teamcity.spec.mainNode", teamcity.Spec.MainNode); err != nil {
		return err
	}
	for idx, secondaryNode := range teamcity.Spec.SecondaryNodes {
		if err := validateRequestsInNode(fmt.Sprintf("teamcity.spec.secondaryNode[%d]", idx), secondaryNode); err != nil {
			return err
		}
	}
	return err
}

func validateResponsibilitiesOfAllNodes(teamcity *TeamCity) (warning string, err error) {
	//it is allowed to have empty responsibilities for all nodes
	if allNodesHaveEmptyResponsibility(teamcity.Spec.MainNode, teamcity.Spec.SecondaryNodes) {
		return "", nil
	}

	//if responsibilities are specified for at least one node, we need to check all of them
	if err = validateMainNodeResponsibilities("teamcity.spec.mainNode", teamcity.Spec.MainNode, validMainNodeResponsibilities, minimumRequiredMainNodeResponsibilities); err != nil {
		return "", err
	}
	for idx, secondaryNode := range teamcity.Spec.SecondaryNodes {
		if err = validateNodeResponsibilities(fmt.Sprintf("teamcity.spec.secondaryNode[%d]", idx), secondaryNode, validSecondaryNodeResponsibilities); err != nil {
			return "", err
		}
	}

	//make sure that all responsibilities are assigned
	if warning := validatePresenceOfAllResponsibilities(allTeamCityResponsibilities, teamcity.Spec.MainNode, teamcity.Spec.SecondaryNodes); warning != "" {
		return warning, err
	}
	return "", nil
}

func validatePresenceOfAllResponsibilities(allResponsibilities []string, mainNode Node, secondaryNodes []Node) string {
	responsibilities := getAllResponsibilitiesFromAllNodes(mainNode, secondaryNodes)
	if !allElementsInOtherSlice(allResponsibilities, responsibilities) {
		return fmt.Sprintf("Not all responsibilities are distributed across the nodes. This may impact functionality of the server. Make sure that the following responsibilities are present in configuration %s", strings.Join(allResponsibilities, ", "))
	}
	return ""
}
func allNodesHaveEmptyResponsibility(mainNode Node, secondaryNodes []Node) bool {
	responsibilities := getAllResponsibilitiesFromAllNodes(mainNode, secondaryNodes)
	return len(responsibilities) == 0
}

func getAllResponsibilitiesFromAllNodes(mainNode Node, secondaryNodes []Node) []string {
	responsibilities := []string{}
	responsibilities = append(responsibilities, mainNode.Spec.Responsibilities...)
	for _, secondaryNode := range secondaryNodes {
		responsibilities = append(responsibilities, secondaryNode.Spec.Responsibilities...)
	}
	return responsibilities
}

func validateMainNodeResponsibilities(objectPath string, node Node, validResponsibilities []string, requiredResponsibilities []string) error {
	responsibilities := node.Spec.Responsibilities
	if len(responsibilities) < 1 {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "responsibilities"),
			ErrorMessage: fmt.Sprintf("Main node cannot have empty responsibilities. Minimum required values are: %s. Valid values are: %s.", strings.Join(requiredResponsibilities, ", "), strings.Join(validResponsibilities, ", ")),
		}
	}
	if !areAllElementsAllowed(responsibilities, validResponsibilities) {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "responsibilities"),
			ErrorMessage: fmt.Sprintf("Main node does not have valid responsibilities. Minimum required values are: %s. Valid values are: %s", strings.Join(requiredResponsibilities, ", "), strings.Join(validResponsibilities, ", ")),
		}
	}
	if !allElementsInOtherSlice(requiredResponsibilities, responsibilities) {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "responsibilities"),
			ErrorMessage: fmt.Sprintf("Main node does not have required responsibilities. Minimum required values are: %s", strings.Join(requiredResponsibilities, ", ")),
		}
	}

	return nil
}

func validateNodeResponsibilities(objectPath string, node Node, validResponsibilities []string) error {
	responsibilities := node.Spec.Responsibilities
	if !areAllElementsAllowed(responsibilities, validResponsibilities) {
		return typed.ValidationError{
			Path:         fmt.Sprintf("%s.%s", objectPath, "responsibilities"),
			ErrorMessage: fmt.Sprintf("Secondary node does not contain valid responsibilities. Valid values are: %s", strings.Join(validResponsibilities, ", ")),
		}
	}

	return nil
}

func areAllElementsAllowed(elements []string, allowed []string) bool {
	allowedMap := make(map[string]bool, len(allowed))
	for _, v := range allowed {
		allowedMap[v] = true
	}

	for _, e := range elements {
		if _, isAllowed := allowedMap[e]; !isAllowed {
			return false
		}
	}

	return true
}

func allElementsInOtherSlice(slice1 []string, slice2 []string) bool {
	m := make(map[string]bool, len(slice2))
	for _, item := range slice2 {
		m[item] = true
	}

	for _, item := range slice1 {
		if _, found := m[item]; !found {
			return false
		}
	}
	return true
}
