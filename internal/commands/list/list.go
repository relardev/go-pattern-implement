package list

import "strings"

func List() string {
	options := []string{
		"prometheus",
		"filegetter",
	}
	return strings.Join(options, "\n") + "\n"
}
