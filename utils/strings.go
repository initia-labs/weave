package utils

import (
	"strings"
)

func TransformFirstWordUpperCase(input string) string {
	words := strings.Fields(input)
	if len(words) > 0 {
		return strings.ToUpper(words[0])
	}
	return ""
}
