package username

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/Qman110101/chunisupport-api/internal/domain/vo"
)

type UserName struct {
	value string
}

// NewUserName はバリデーション付きで新しい UserName を作成します
func NewUserName(value string) (UserName, error) {
	if err := validateUserName(value); err != nil {
		return UserName{}, err
	}
	return UserName{value: value}, nil
}

// MustNewUserName はバリデーションなしで新しい UserName を作成します
// バリデーションエラーが発生した場合はパニックします
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
		return errors.New("username cannot be empty")
	}
	if len(value) < 5 {
		return errors.New("username must be at least 5 characters")
	}
	if len(value) > 50 {
		return errors.New("username must be 50 characters or less")
	}
	matched, err := regexp.MatchString("^[a-z0-9]+$", value)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("username can only contain lowercase letters and numbers")
	}
	return nil
}
