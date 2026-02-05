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

// NewPlayerName はバリデーション付きで新しいPlayerNameを作成します
// プレイヤー名は全角8文字以下である必要があります
func NewPlayerName(value string) (PlayerName, error) {
	if err := validatePlayerName(value); err != nil {
		return PlayerName{}, err
	}
	return PlayerName{value: value}, nil
}

// MustNewPlayerName はバリデーションなしで新しいPlayerNameを作成します
// バリデーションエラーが発生した場合はパニックします
// 警告: テストコード専用。本番コードでは使用禁止。
// 既にバリデーション済みの値を使用する場合にのみ使用してください
func MustNewPlayerName(value string) PlayerName {
	playerName, err := NewPlayerName(value)
	if err != nil {
		panic(err)
	}
	return playerName
}

// String は PlayerName の文字列値を返します
func (p PlayerName) String() string {
	return p.value
}

// Value は database/sql の driver.Valuer インターフェースを実装します
func (p PlayerName) Value() (driver.Value, error) {
	return p.value, nil
}

func (p *PlayerName) Scan(src any) error {
	if src == nil {
		// DBからnullが来た場合は空のPlayerNameを設定
		// ただしバリデーションは行わない（DB上のデータは信頼する）
		*p = PlayerName{value: ""}
		return nil
	}

	s, err := vo.ToString(src)
	if err != nil {
		return err
	}

	// DBから取得した値はバリデーション済みとみなす
	// 空文字の場合もそのまま設定
	if s == "" {
		*p = PlayerName{value: ""}
		return nil
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
	// エスケープを適切に処理するためにjson.Marshalを使用
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

// validatePlayerName はプレイヤー名が全角8文字以下であることを検証します
func validatePlayerName(value string) error {
	if value == "" {
		return errors.New("player name cannot be empty")
	}

	// for _, r := range value {
	// 	kind := width.LookupRune(r).Kind()
	// 	if kind != width.EastAsianFullwidth && kind != width.EastAsianWide {
	// 		return errors.New("player name must contain only full-width characters")
	// 	}
	// }

	// 全角文字数をカウント（UTF-8のルーン数をカウント）
	runeCount := utf8.RuneCountInString(value)
	if runeCount > 8 {
		return errors.New("player name must be 8 characters or less")
	}

	return nil
}
