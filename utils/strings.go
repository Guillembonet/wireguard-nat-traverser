package utils

import "strings"

func GetQuery(message string) []string {
	return strings.Split(strings.ReplaceAll(message, "\n", ""), " ")
}
