package entity

import (
	"errors"
	"regexp"
)

// XAccount はユーザーが連携している X アカウントの識別情報を表します。
// X のユーザー名は変更できるため、変わらない数値IDと組で保持し、表示側が現在の連携先と照合できるようにします。
type XAccount struct {
	userID   int
	xUserID  string
	username string
}

var (
	ErrXAccountUserIDInvalid   = errors.New("x account user_id is invalid")
	ErrXAccountXUserIDInvalid  = errors.New("x account x_user_id is invalid")
	ErrXAccountUsernameInvalid = errors.New("x account username is invalid")
)

// X のユーザーIDは64bit整数の10進表記、ユーザー名は英数字とアンダースコアです。
// 保存列の長さを超える値や想定外の文字は、X 側の仕様変更か不正な応答とみなして保存しません。
var (
	xUserIDPattern   = regexp.MustCompile(`^[0-9]{1,20}$`)
	xUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,50}$`)
)

// NewXAccount は X から取得した識別情報、または永続化済みの値を検証して XAccount を生成します。
func NewXAccount(userID int, xUserID string, username string) (*XAccount, error) {
	if userID <= 0 {
		return nil, ErrXAccountUserIDInvalid
	}
	if !xUserIDPattern.MatchString(xUserID) {
		return nil, ErrXAccountXUserIDInvalid
	}
	if !xUsernamePattern.MatchString(username) {
		return nil, ErrXAccountUsernameInvalid
	}

	return &XAccount{
		userID:   userID,
		xUserID:  xUserID,
		username: username,
	}, nil
}

// UserID は所有ユーザーIDを返します。
func (a *XAccount) UserID() int {
	return a.userID
}

// XUserID は X の数値ユーザーIDを返します。
func (a *XAccount) XUserID() string {
	return a.xUserID
}

// Username は @ を除いた X のユーザー名を返します。
func (a *XAccount) Username() string {
	return a.username
}
