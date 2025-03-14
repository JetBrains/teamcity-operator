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
	if warn, err := validateCommonFields(instance); err != nil {
		return warn, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (instance *TeamCity) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	teamcitylog.Info("validate update", "name", instance.Name)
	warn, err := validateCommonFields(instance)
	if err != nil {
		return nil, err
	}
	return warn, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (instance *TeamCity) ValidateDelete() (admission.Warnings, error) {
	teamcitylog.Info("validate delete", "name", instance.Name)

	return nil, nil
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
	if responsibilityWarning, err := validateResponsibilitiesOfAllNodes(teamcity); err != nil || responsibilityWarning != "" {
		return admission.Warnings{responsibilityWarning}, err
	}
	return nil, nil
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
	return nil
}

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
