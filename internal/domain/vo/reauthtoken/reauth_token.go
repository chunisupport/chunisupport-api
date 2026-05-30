package reauthtoken

import (
	"errors"
	"strings"
)

var (
	// ErrEmpty は再認証トークンが空文字の場合に返されます。
	ErrEmpty = errors.New("reauth token is required")
)

// ReauthToken は再認証トークンを表す値オブジェクトです。
type ReauthToken struct {
	value string
}

// New は正規化とバリデーションを行った ReauthToken を生成します。
func New(value string) (ReauthToken, error) {
	normalizedValue := strings.TrimSpace(value)
	if normalizedValue == "" {
		return ReauthToken{}, ErrEmpty
	}

	return ReauthToken{value: normalizedValue}, nil
}

// MustNew はテスト用途の ReauthToken を生成します。
func MustNew(value string) ReauthToken {
	token, err := New(value)
	if err != nil {
		panic(err)
	}

	return token
}

// String は正規化済みの文字列表現を返します。
func (t ReauthToken) String() string {
	return t.value
}
