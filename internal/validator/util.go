package validator

import "strings"

func ByteArrayToLineStringArray(array []uint8) []string {
	return deleteEmpty(strings.Split(string(array), "\n"))
}

func deleteEmpty(strarr []string) []string {
	var arr []string
	for _, str := range strarr {
		if str != "" {
			arr = append(arr, str)
		}
	}
	return arr

}
