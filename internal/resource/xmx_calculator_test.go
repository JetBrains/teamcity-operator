package resource

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("XmxValueCalculator", func() {
	It("returns the requested percentage of memory", func() {
		Expect(XmxValueCalculator(90, 1000)).To(Equal("900"))
	})
	It("rounds down fractional results", func() {
		Expect(XmxValueCalculator(33, 10)).To(Equal("3"))
	})
})
