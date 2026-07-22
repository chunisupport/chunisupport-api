package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenModel はデータベース用のAPITokenモデルです。
type APITokenModel struct {
	ID          uint64     `db:"id"`
	UserID      int        `db:"user_id"`
	Name        string     `db:"name"`
	HashedToken string     `db:"hashed_token"`
	TokenPrefix *string    `db:"token_prefix"`
	LastUsedAt  *time.Time `db:"last_used_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

// ToEntity は永続化モデルを検証済みのドメインエンティティへ変換します。
func (m *APITokenModel) ToEntity() (*entity.APIToken, error) {
	return entity.RestoreAPIToken(m.ID, m.UserID, m.Name, m.HashedToken, m.TokenPrefix, m.LastUsedAt, m.CreatedAt)
}

// FromAPITokenEntity はドメインエンティティを永続化モデルへ変換します。
func FromAPITokenEntity(e *entity.APIToken) *APITokenModel {
	return &APITokenModel{
		ID:          e.ID,
		UserID:      e.UserID,
		Name:        e.Name.String(),
		HashedToken: e.HashedToken,
		TokenPrefix: e.TokenPrefix,
		LastUsedAt:  e.LastUsedAt,
		CreatedAt:   e.CreatedAt,
	}
}
