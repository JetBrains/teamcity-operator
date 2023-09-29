package validator

import "strings"

func ByteArrayToLineStringArray(array []uint8) []string {
	return strings.Split(string(array), "\n")
}
