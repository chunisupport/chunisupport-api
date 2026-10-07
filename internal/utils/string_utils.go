package utils

import "strings"

// EscapeLike はSQLのLIKE句で使う文字列の '%', '_', '\' をエスケープします。
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
