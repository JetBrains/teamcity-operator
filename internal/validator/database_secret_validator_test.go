package validator

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
	"regexp"
)

var _ = Describe("Database Secret Validator", func() {
	Context("Secret object with valid configuration", func() {
		It("produces no error on valid object", func() {
			secret := v12.Secret{
				Data: map[string][]byte{
					"database.properties": []byte("connectionUrl=jdbc:mysql://mysql.default:3306/teamcity\nconnectionProperties.user=root\nconnectionProperties.password=password"),
				},
			}
			validator := DatabaseSecretValidator{&secret}
			err := validator.ValidateObject()
			Expect(err).To(BeNil())
		})
		It("produces no error on valid object with extra property", func() {
			secret := v12.Secret{
				Data: map[string][]byte{
					"database.properties": []byte("connectionUrl=jdbc:mysql://mysql.default:3306/teamcity\nconnectionProperties.user=root\nconnectionProperties.password=password\nconnectionProperties.socketTimeout=900000"),
				},
			}
			validator := DatabaseSecretValidator{&secret}
			err := validator.ValidateObject()
			Expect(err).To(BeNil())
		})
		Context("Secret object with with invalid configuration", func() {
			It("produces error on valid object with incorrect key", func() {
				secret := v12.Secret{
					Data: map[string][]byte{
						"database-properties": []byte("connectionUrl=jdbc:mysql://mysql.default:3306/teamcity\nconnectionProperties.user=root\nconnectionProperties.password=password\nconnectionProperties.socketTimeout=900000"),
					},
				}
				validator := DatabaseSecretValidator{&secret}
				err := validator.ValidateObject()
				Expect(err).ToNot(BeNil())

				errorMatched, _ := regexp.MatchString("Expected key [A-Za-z]+\\.[A-Za-z]+ is not present in database secret", err.Error())
				Expect(errorMatched).To(Equal(true))
			})
			It("produces error on missing required properties", func() {
				secret := v12.Secret{
					Data: map[string][]byte{
						"database.properties": []byte("connectionUrl=jdbc:mysql://mysql.default:3306/teamcity"),
					},
				}
				validator := DatabaseSecretValidator{&secret}
				err := validator.ValidateObject()
				Expect(err).ToNot(BeNil())

				errorMatched, _ := regexp.MatchString("Required key [A-Za-z]+\\.[A-Za-z]+ is not provided in database secret", err.Error())
				Expect(errorMatched).To(Equal(true))
			})
			It("produces error on object with missing value", func() {
				secret := v12.Secret{
					Data: map[string][]byte{
						"database.properties": []byte("connectionUrl=\nconnectionProperties.user=root\nconnectionProperties.password=password\nconnectionProperties.socketTimeout=900000"),
					},
				}
				validator := DatabaseSecretValidator{&secret}
				err := validator.ValidateObject()
				Expect(err).ToNot(BeNil())

				errorMatched, _ := regexp.MatchString("Value is not provided for key [A-Za-z]+ in database secret", err.Error())
				Expect(errorMatched).To(Equal(true))

			})
		})

	})

})
