package validator

import "sigs.k8s.io/controller-runtime/pkg/client"

type TeamCityDependencyValidator struct {
}

type Validator interface {
	IsValid() bool
	Validate(object client.Object) error
}

func (validator *TeamCityDependencyValidator) Validators() []Validator {

	validators := []Validator{
		validator.DatabaseSecretValidator(),
	}
	return validators
}
