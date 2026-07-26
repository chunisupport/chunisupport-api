// Package maintenancecomment はメンテナンス中に公開するコメントの値オブジェクトを提供します。
package maintenancecomment

import (
	"errors"
	"strings"
	"unicode"
)

// MaxLength はメンテナンスコメントに許可するUnicodeコードポイント数です。
const MaxLength = 1000

var (
	// ErrRequired はメンテナンス開始用コメントが空の場合に返されます。
	ErrRequired = errors.New("maintenance comment is required")
	// ErrTooLong はコメントが最大文字数を超えた場合に返されます。
	ErrTooLong = errors.New("maintenance comment is too long")
	// ErrControlCharacter は改行以外の制御文字を含む場合に返されます。
	ErrControlCharacter = errors.New("maintenance comment contains a control character")
)

// MaintenanceComment は正規化・検証済みのメンテナンスコメントです。
type MaintenanceComment struct {
	value string
}

// NewMaintenanceComment はメンテナンス開始に使用できる非空コメントを生成します。
func NewMaintenanceComment(value string) (MaintenanceComment, error) {
	comment, err := normalize(value)
	if err != nil {
		return MaintenanceComment{}, err
	}
	if comment == "" {
		return MaintenanceComment{}, ErrRequired
	}
	return MaintenanceComment{value: comment}, nil
}

// RestoreMaintenanceComment はDBに保存されたコメントを復元します。
// メンテナンス無効時の正しい永続状態を表現するため、空文字を許可します。
func RestoreMaintenanceComment(value string) (MaintenanceComment, error) {
	comment, err := normalize(value)
	if err != nil {
		return MaintenanceComment{}, err
	}
	return MaintenanceComment{value: comment}, nil
}

// String は正規化済みコメントを返します。
func (c MaintenanceComment) String() string {
	return c.value
}

// IsEmpty はコメントが空かを返します。
func (c MaintenanceComment) IsEmpty() bool {
	return c.value == ""
}

func normalize(value string) (string, error) {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	for _, r := range normalized {
		if r != '\n' && unicode.IsControl(r) {
			return "", ErrControlCharacter
		}
	}
	normalized = strings.TrimSpace(normalized)
	if len([]rune(normalized)) > MaxLength {
		return "", ErrTooLong
	}
	return normalized, nil
}
