package predicate

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestTeamcityEventPredicates(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Predicate Suite")
}
