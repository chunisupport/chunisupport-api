package passwordhash

import (
	"database/sql/driver"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo"
)

// PasswordHash はパスワードハッシュの値オブジェクトです。
type PasswordHash string

// NewPasswordHash は新しい PasswordHash を作成します
func NewPasswordHash(value string) (PasswordHash, error) {
	if value == "" {
		return "", errors.New("password hash cannot be empty")
	}
	return PasswordHash(value), nil
}

// NewEmptyPasswordHash はパスワードなし状態を表す空の PasswordHash を返します。
// Firebase認証のみで登録したユーザーなど、パスワードを持たないユーザーに使用します。
func NewEmptyPasswordHash() PasswordHash {
	return PasswordHash("")
}

// MustNewPasswordHash はテストや固定値用のヘルパーです。
// 警告: テストコード専用。本番コードでは使用禁止。
func MustNewPasswordHash(value string) PasswordHash {
	ph, err := NewPasswordHash(value)
	if err != nil {
		panic(err)
	}
	return ph
}

// String は PasswordHash の文字列表現を返します。
func (p PasswordHash) String() string {
	return string(p)
}

// Value は driver.Valuer インターフェースを実装します。
func (p PasswordHash) Value() (driver.Value, error) {
	return string(p), nil
}

// Scan は sql.Scanner インターフェースを実装します。
func (p *PasswordHash) Scan(src any) error {
	if src == nil {
		*p = ""
		return nil
	}

	s, err := vo.ToString(src)
	if err != nil {
		return err
	}

	*p = PasswordHash(s)
	return nil
}
