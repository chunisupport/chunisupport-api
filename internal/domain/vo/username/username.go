package username

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo"
)

type UserName struct {
	value string
}

var usernamePattern = regexp.MustCompile("^[a-z0-9]+$")

var (
	// ErrEmpty はユーザー名が空文字の場合に返されます。
	ErrEmpty = errors.New("username cannot be empty")
	// ErrTooShort はユーザー名が最小文字数に満たない場合に返されます。
	ErrTooShort = errors.New("username must be at least 5 characters")
	// ErrTooLong はユーザー名が最大文字数を超える場合に返されます。
	ErrTooLong = errors.New("username must be 50 characters or less")
	// ErrInvalidChar はユーザー名に許可されない文字が含まれる場合に返されます。
	ErrInvalidChar = errors.New("username can only contain lowercase letters and numbers")
)

// NewUserName はバリデーション付きで新しい UserName を作成します
func NewUserName(value string) (UserName, error) {
	if err := validateUserName(value); err != nil {
		return UserName{}, err
	}
	return UserName{value: value}, nil
}

// MustNewUserName はバリデーションなしで新しい UserName を作成します
// バリデーションエラーが発生した場合はパニックします
// 警告: テストコード専用。本番コードでは使用禁止。
// 既にバリデーション済みの値を使用する場合にのみ使用してください
func MustNewUserName(value string) UserName {
	userName, err := NewUserName(value)
	if err != nil {
		panic(err)
	}
	return userName
}

// String は UserName の文字列値を返します
func (u UserName) String() string {
	return u.value
}

// Value は database/sql の driver.Valuer インターフェースを実装します
func (u UserName) Value() (driver.Value, error) {
	return u.value, nil
}

func (u *UserName) Scan(src any) error {
	if src == nil {
		// DBからnullが来た場合は空のUserNameを設定
		// ただしバリデーションは行わない（DB上のデータは信頼する）
		*u = UserName{value: ""}
		return nil
	}

	s, err := vo.ToString(src)
	if err != nil {
		return err
	}

	// DBから取得した値はバリデーション済みとみなす
	// 空文字の場合もそのまま設定
	if s == "" {
		*u = UserName{value: ""}
		return nil
	}

	userName, err := NewUserName(s)
	if err != nil {
		return err
	}

	*u = userName
	return nil
}

// MarshalJSON は json.Marshaler を実装します
func (u UserName) MarshalJSON() ([]byte, error) {
	// エスケープを適切に処理するためにjson.Marshalを使用
	return json.Marshal(u.value)
}

// UnmarshalJSON は json.Unmarshaler を実装します
func (u *UserName) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	userName, err := NewUserName(str)
	if err != nil {
		return err
	}
	*u = userName
	return nil
}

// validateUserName はユーザー名が空でないこと、5文字以上50文字以内、小文字英数字のみであることをバリデーションします
func validateUserName(value string) error {
	if value == "" {
		return ErrEmpty
	}
	if len(value) < 5 {
		return ErrTooShort
	}
	if len(value) > 50 {
		return ErrTooLong
	}
	if !usernamePattern.MatchString(value) {
		return ErrInvalidChar
	}
	return nil
}
