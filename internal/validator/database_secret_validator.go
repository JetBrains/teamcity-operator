package validator

import "sigs.k8s.io/controller-runtime/pkg/client"

type DatabaseSecretValidator struct {
	*TeamCityDependencyValidator
}

func (validator *TeamCityDependencyValidator) DatabaseSecretValidator() *DatabaseSecretValidator {
	return &DatabaseSecretValidator{validator}
}

func (validator DatabaseSecretValidator) IsValid() bool {
	return true
}

func (validator DatabaseSecretValidator) Validate(object client.Object) error {
	return nil
}
