package playername

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo"
)

type PlayerName struct {
	value string
}

// NewPlayerName は空文字・半角英数字・半角カタカナを拒否し、Unicodeコードポイント数が8以下の名前を生成します。
func NewPlayerName(value string) (PlayerName, error) {
	if err := validatePlayerName(value); err != nil {
		return PlayerName{}, err
	}
	return PlayerName{value: value}, nil
}

// String は PlayerName の文字列値を返します
func (p PlayerName) String() string {
	return p.value
}

// Value は database/sql の driver.Valuer インターフェースを実装します
func (p PlayerName) Value() (driver.Value, error) {
	return p.value, nil
}

// Scan はDBから取得した値を検証し、PlayerNameを復元します。
// 不正なDB値から値オブジェクトの不変条件が壊れないよう、NULLもエラーとして扱います。
func (p *PlayerName) Scan(src any) error {
	if src == nil {
		return validatePlayerName("")
	}

	s, err := vo.ToString(src)
	if err != nil {
		return err
	}

	playerName, err := NewPlayerName(s)
	if err != nil {
		return err
	}

	*p = playerName
	return nil
}

// MarshalJSON は json.Marshaler を実装します
func (p PlayerName) MarshalJSON() ([]byte, error) {
	// 手動でJSON文字列を組み立てず、json.Marshal にエスケープを任せます。
	return json.Marshal(p.value)
}

// UnmarshalJSON は json.Unmarshaler を実装します
func (p *PlayerName) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	playerName, err := NewPlayerName(str)
	if err != nil {
		return err
	}
	*p = playerName
	return nil
}

// validatePlayerName は半角英数字・半角カタカナを拒否し、Unicodeコードポイント数を8文字以下に制限します。
func validatePlayerName(value string) error {
	if value == "" {
		return errors.New("player name cannot be empty")
	}

	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return errors.New("player name must not contain half-width alphanumeric characters")
		}
		if r >= 0xFF61 && r <= 0xFF9F {
			return errors.New("player name must not contain half-width katakana")
		}
	}

	// 全角文字数をカウント（UTF-8のルーン数をカウント）
	runeCount := utf8.RuneCountInString(value)
	if runeCount > 8 {
		return errors.New("player name must be 8 characters or less")
	}

	return nil
}
