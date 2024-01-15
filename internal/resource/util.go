package resource

import (
	"sort"
)

func SortKeysAlphabeticallyInMap(input map[string]string) []string {
	s := make([]string, 0, len(input))
	for k, _ := range input {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}
