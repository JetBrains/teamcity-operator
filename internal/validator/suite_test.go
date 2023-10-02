package validator

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator Suite")
}
