package resource

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Context("Sort", func() {
		It("processes map with unsorted keys correctly", func() {
			testMap := map[string]string{
				"a": "1",
				"c": "2",
				"b": "3",
			}
			expected := []string{"a", "b", "c"}
			sortedKeys := SortKeysAlphabeticallyInMap(testMap)
			Expect(expected).To(Equal(sortedKeys))
		})
		It("returns an empty slice for an empty map", func() {
			sortedKeys := SortKeysAlphabeticallyInMap(map[string]string{})
			Expect(sortedKeys).To(BeEmpty())
		})
		It("returns an empty slice for a nil map", func() {
			sortedKeys := SortKeysAlphabeticallyInMap(nil)
			Expect(sortedKeys).To(BeEmpty())
		})
	})
})
