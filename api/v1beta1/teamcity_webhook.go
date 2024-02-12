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
)

// log is for logging in this package.
var teamcitylog = logf.Log.WithName("teamcity-resource")

func (r *TeamCity) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-jetbrains-com-v1beta1-teamcity,mutating=true,failurePolicy=fail,sideEffects=None,groups=jetbrains.com,resources=teamcities,verbs=create;update,versions=v1beta1,name=mv1beta1teamcity.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &TeamCity{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *TeamCity) Default() {
	teamcitylog.Info("default", "name", r.Name)

}

//+kubebuilder:webhook:path=/validate-jetbrains-com-v1beta1-teamcity,mutating=false,failurePolicy=fail,sideEffects=None,groups=jetbrains.com,resources=teamcities,verbs=create;update,versions=v1beta1,name=vv1beta1teamcity.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &TeamCity{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TeamCity) ValidateCreate() (admission.Warnings, error) {
	teamcitylog.Info("validate create", "name", r.Name)
	if warn, err := validateCommonFields(r); err != nil {
		return warn, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TeamCity) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	teamcitylog.Info("validate update", "name", r.Name)
	if warn, err := validateCommonFields(r); err != nil {
		return warn, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TeamCity) ValidateDelete() (admission.Warnings, error) {
	teamcitylog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func validateCommonFields(teamcity *TeamCity) (admission.Warnings, error) {
	if err := validateReplicas(teamcity); err != nil {
		return nil, err
	}
	if err := validateRequests(teamcity); err != nil {
		return nil, err
	}
	if err := validateXmxPercentage(teamcity); err != nil {
		return nil, err
	}
	if err := validateAllCustomPersistentVolumeClaimsInObject(teamcity); err != nil {
		return nil, err
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

func validateReplicas(teamcity *TeamCity) (err error) {
	if *teamcity.Spec.Replicas > 1 {
		return typed.ValidationError{
			Path:         "teamcity.spec.replicas",
			ErrorMessage: "Replicas value cannot be greater than 1",
		}
	}
	return nil
}

func validateRequests(teamcity *TeamCity) (err error) {
	if len(teamcity.Spec.Requests.Memory().String()) <= 0 {
		return typed.ValidationError{
			Path:         "teamcity.spec.requests.memory",
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
