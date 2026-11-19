package jsonquery

import "github.com/tidwall/gjson"

func Value(document, path string) string {
	return gjson.Get(document, path).String()
}
