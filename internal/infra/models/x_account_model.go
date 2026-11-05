package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// XAccountModel はデータベース用の XAccount モデルです。
type XAccountModel struct {
	UserID    int    `db:"user_id"`
	XUserID   string `db:"x_user_id"`
	XUsername string `db:"x_username"`
}

// ToEntity は保存済みの行を XAccount へ復元します。
func (m *XAccountModel) ToEntity() (*entity.XAccount, error) {
	return entity.NewXAccount(m.UserID, m.XUserID, m.XUsername)
}
