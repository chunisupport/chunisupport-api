package displayid

import (
	"errors"
	"regexp"
)

// DisplayID は楽曲の表示用IDを表すValue Object
// 16進数16文字の固定長文字列(CHAR(16))
// 外部バッチで生成されたIDをバリデーションするために使用
type DisplayID string

var (
	// 16進数16文字の正規表現パターン
	displayIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

	// エラー定義
	ErrInvalidDisplayIDFormat = errors.New("display ID must be exactly 16 hexadecimal characters (lowercase)")
)

// NewDisplayID は文字列からDisplayIDを生成し、バリデーションを行います
// ID生成ロジックは外部バッチ側で実装されるため、ここではバリデーションのみを行います
func NewDisplayID(id string) (DisplayID, error) {
	if !displayIDPattern.MatchString(id) {
		return "", ErrInvalidDisplayIDFormat
	}
	return DisplayID(id), nil
}

// String はDisplayIDを文字列として返します
func (d DisplayID) String() string {
	return string(d)
}

// IsValid はDisplayIDが有効な形式かを確認します
func (d DisplayID) IsValid() bool {
	return displayIDPattern.MatchString(string(d))
}
