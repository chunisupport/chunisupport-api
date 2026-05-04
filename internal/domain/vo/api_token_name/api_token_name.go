package api_token_name

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const maxLength = 15

var (
	ErrTooLong = errors.New("api token name must be 15 characters or less")
)

// Normalize はAPIトークン名を正規化し、ドメインルールに従って検証します。
func Normalize(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", nil
	}
	if utf8.RuneCountInString(normalized) > maxLength {
		return "", ErrTooLong
	}
	return normalized, nil
}
