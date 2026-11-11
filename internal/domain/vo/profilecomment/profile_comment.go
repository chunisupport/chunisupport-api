package profilecomment

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ProfileComment は前後の空白を除去した1行のプロフィールコメントです。
type ProfileComment struct{ value string }

var (
	ErrEmpty       = errors.New("profile comment cannot be empty")
	ErrTooLong     = errors.New("profile comment must be 100 characters or less")
	ErrInvalidChar = errors.New("profile comment contains invalid characters")
)

// NewProfileComment は保存時にも復元時にも同じ制約を適用します。
// 空は未設定を表すため、値オブジェクトとしては生成しません。
func NewProfileComment(value string) (ProfileComment, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ProfileComment{}, ErrEmpty
	}
	if !utf8.ValidString(value) {
		return ProfileComment{}, ErrInvalidChar
	}
	if utf8.RuneCountInString(value) > MaxLength {
		return ProfileComment{}, ErrTooLong
	}
	for _, r := range value {
		// Unicodeの行・段落区切りも1行表示を壊すため拒否します。
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return ProfileComment{}, ErrInvalidChar
		}
	}
	return ProfileComment{value: value}, nil
}

// String は正規化済みのコメントを返します。
func (c ProfileComment) String() string { return c.value }
