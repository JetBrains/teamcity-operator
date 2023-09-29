package validator

import (
	"fmt"
	v12 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const EXPECTED_DATABASE_SECRET_KEY = "database.properties"

var requiredDatabaseSecretEntries = []string{"connectionUrl", "connectionProperties.user", "connectionProperties.password"}

type DatabaseSecretValidator struct {
	client.Object
}

func (validator *TeamCityDependencyValidator) DatabaseSecretValidator(object client.Object) *DatabaseSecretValidator {
	return &DatabaseSecretValidator{object}
}

func (validator DatabaseSecretValidator) ValidateObject() error {
	databaseSecret := validator.Object.(*v12.Secret)
	secretBody := databaseSecret.Data
	databasePropertiesContent, ok := secretBody[EXPECTED_DATABASE_SECRET_KEY]
	var databasePropertyKeys []string
	if !ok {
		return fmt.Errorf("Expected key %s is not present in database secret", EXPECTED_DATABASE_SECRET_KEY)
	}
	for _, line := range ByteArrayToLineStringArray(databasePropertiesContent) {
		lineSplit := strings.Split(line, "=")
		if len(lineSplit) < 1 {
			return fmt.Errorf("Value is not provided for key %s un database secret", lineSplit[0])
		}
		databasePropertyKey := lineSplit[0]
		databasePropertyKeys = append(databasePropertyKeys, databasePropertyKey)
	}
	missingKeys := findMissingKeys(requiredDatabaseSecretEntries, databasePropertyKeys)

	for _, key := range missingKeys {
		if validator.isRequiredKey(key) {
			return fmt.Errorf("Required key %s is not provided in database secret", key)
		}
	}

	return nil
}

func (validator DatabaseSecretValidator) isRequiredKey(line string) bool {
	return slices.Contains(requiredDatabaseSecretEntries, line)
}

func findMissingKeys(slice1 []string, slice2 []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff = append(diff, s1)
			}
		}
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}

	return diff
}
