// Package apitokenname はAPIトークン名の値オブジェクトを提供します。
package apitokenname

import (
	"errors"
	"strings"
	"unicode"
)

const MaxLength = 50

var ErrInvalidAPITokenName = errors.New("invalid API token name")

// APITokenName は検証済みのAPIトークン名です。
type APITokenName struct {
	value string
}

// NewAPITokenName は前後の空白を除去し、表示可能な1〜50文字の名前を生成します。
func NewAPITokenName(value string) (APITokenName, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len([]rune(trimmed)) > MaxLength {
		return APITokenName{}, ErrInvalidAPITokenName
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return APITokenName{}, ErrInvalidAPITokenName
		}
	}
	return APITokenName{value: trimmed}, nil
}

// String はAPIトークン名を返します。
func (n APITokenName) String() string {
	return n.value
}
