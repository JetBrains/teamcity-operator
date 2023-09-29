package validator

type TeamCityDependencyValidator struct {
}

type Validator interface {
	ValidateObject() error
}
