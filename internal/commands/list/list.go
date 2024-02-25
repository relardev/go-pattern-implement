package list

import "strings"

func List() string {
	options := []string{
		"prometheus",
		"slog",
	}
	return strings.Join(options, "\n")
}
