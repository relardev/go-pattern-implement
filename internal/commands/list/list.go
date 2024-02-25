package list

import "strings"

func List() string {
	options := []string{
		"prometheus",
	}
	return strings.Join(options, "\n")
}
