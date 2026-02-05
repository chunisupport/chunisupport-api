package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecoveryCodeModel はデータベース用のリカバリーコードモデルです。
type RecoveryCodeModel struct {
	ID        uint32    `db:"id"`
	UserID    int       `db:"user_id"`
	CodeHash  []byte    `db:"code_hash"`
	CreatedAt time.Time `db:"created_at"`
}

func (m *RecoveryCodeModel) ToEntity() *entity.RecoveryCode {
	return &entity.RecoveryCode{
		ID:        m.ID,
		UserID:    m.UserID,
		CodeHash:  m.CodeHash,
		CreatedAt: m.CreatedAt,
	}
}

func FromRecoveryCodeEntity(e *entity.RecoveryCode) *RecoveryCodeModel {
	return &RecoveryCodeModel{
		ID:        e.ID,
		UserID:    e.UserID,
		CodeHash:  e.CodeHash,
		CreatedAt: e.CreatedAt,
	}
}
