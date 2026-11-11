package entity

import (
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment"
)

// UserProfile はUser集約とは独立して保存するユーザーのプロフィールです。
type UserProfile struct {
	userID  int
	comment profilecomment.ProfileComment
}

var ErrUserProfileUserIDInvalid = errors.New("user profile user_id is invalid")

// NewUserProfile は未設定のコメントや不正な所有者を持つ集約の生成を防ぎます。
func NewUserProfile(userID int, comment profilecomment.ProfileComment) (*UserProfile, error) {
	if userID <= 0 {
		return nil, ErrUserProfileUserIDInvalid
	}
	if _, err := profilecomment.NewProfileComment(comment.String()); err != nil {
		return nil, err
	}
	return &UserProfile{userID: userID, comment: comment}, nil
}

// UserID は所有ユーザーIDを返します。
func (p *UserProfile) UserID() int { return p.userID }

// Comment は検証済みのコメントを返します。
func (p *UserProfile) Comment() profilecomment.ProfileComment { return p.comment }
