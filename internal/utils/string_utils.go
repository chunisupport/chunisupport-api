package utils

import "strings"

// EscapeLike escapes special characters in a string for use in a SQL LIKE clause.
// It escapes '%', '_', and '\'.
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
