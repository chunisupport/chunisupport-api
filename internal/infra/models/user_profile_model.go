package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment"
)

// UserProfileModel はプロフィールの永続化専用モデルです。
type UserProfileModel struct {
	UserID  int    `db:"user_id"`
	Comment string `db:"comment"`
}

// ToEntity は保存済みの値にもドメインの制約を適用して復元します。
func (m *UserProfileModel) ToEntity() (*entity.UserProfile, error) {
	comment, err := profilecomment.NewProfileComment(m.Comment)
	if err != nil {
		return nil, err
	}
	return entity.NewUserProfile(m.UserID, comment)
}

// FromEntity は保存に必要な値だけをモデルに写します。
func (m *UserProfileModel) FromEntity(profile *entity.UserProfile) {
	m.UserID = profile.UserID()
	m.Comment = profile.Comment().String()
}
