package util

import (
	"strings"
)

func StringArray(str string) []string {
	return strings.FieldsFunc(str, func(r rune) bool {
		return r == ':' || r == ';'
	})
}
