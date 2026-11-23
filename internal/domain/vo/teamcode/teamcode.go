// Package teamcode はCHUNITHMのチームコードを表す値オブジェクトを提供します。
package teamcode

import (
	"errors"
	"regexp"
)

// TeamCode はCHUNITHM-NETのチームTOPに表示されるチームコードです。
// チーム名は変更できるため、チームを一意に識別する値としてチームコードを使います。
type TeamCode struct {
	value string
}

var (
	// チームコードは大文字英数字6桁の固定長です。
	teamCodePattern = regexp.MustCompile(`^[A-Z0-9]{6}$`)

	// ErrInvalidTeamCode はチームコードの形式が不正な場合のエラーです。
	ErrInvalidTeamCode = errors.New("team code must be exactly 6 uppercase alphanumeric characters")
)

// NewTeamCode は文字列を検証してチームコードを生成します。
// 小文字や前後の空白は取得処理の不具合を隠さないよう正規化せずに拒否します。
func NewTeamCode(value string) (TeamCode, error) {
	if !teamCodePattern.MatchString(value) {
		return TeamCode{}, ErrInvalidTeamCode
	}
	return TeamCode{value: value}, nil
}

// String はチームコードを文字列として返します。
func (c TeamCode) String() string {
	return c.value
}
