package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenModel はデータベース用のAPITokenモデルです。
type APITokenModel struct {
	ID          int64     `db:"id"`
	UserID      int       `db:"user_id"`
	Name        string    `db:"name"`
	HashedToken string    `db:"hashed_token"`
	CreatedAt   time.Time `db:"created_at"`
}

func (m *APITokenModel) ToEntity() *entity.APIToken {
	return &entity.APIToken{
		ID:          m.ID,
		UserID:      m.UserID,
		Name:        m.Name,
		HashedToken: m.HashedToken,
		CreatedAt:   m.CreatedAt,
	}
}

func FromAPITokenEntity(e *entity.APIToken) *APITokenModel {
	return &APITokenModel{
		ID:          e.ID,
		UserID:      e.UserID,
		Name:        e.Name,
		HashedToken: e.HashedToken,
		CreatedAt:   e.CreatedAt,
	}
}
