package models

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// SessionModel はデータベース用のSessionモデルです。
type SessionModel struct {
	ID        string    `db:"id"`
	UserID    int       `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// ToEntity はSessionModelをentity.Sessionに変換します。
func (m *SessionModel) ToEntity() *entity.Session {
	return &entity.Session{
		ID:        m.ID,
		UserID:    m.UserID,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

// FromSessionEntity はentity.SessionをSessionModelに変換します。
func FromSessionEntity(e *entity.Session) *SessionModel {
	return &SessionModel{
		ID:        e.ID,
		UserID:    e.UserID,
		ExpiresAt: e.ExpiresAt,
		CreatedAt: e.CreatedAt,
	}
}

// APITokenModel はデータベース用のAPITokenモデルです。
type APITokenModel struct {
	ID          int64     `db:"id"`
	UserID      int       `db:"user_id"`
	HashedToken string    `db:"hashed_token"`
	CreatedAt   time.Time `db:"created_at"`
}

// ToEntity はAPITokenModelをentity.APITokenに変換します。
func (m *APITokenModel) ToEntity() *entity.APIToken {
	return &entity.APIToken{
		ID:          m.ID,
		UserID:      m.UserID,
		HashedToken: m.HashedToken,
		CreatedAt:   m.CreatedAt,
	}
}

// FromAPITokenEntity はentity.APITokenをAPITokenModelに変換します。
func FromAPITokenEntity(e *entity.APIToken) *APITokenModel {
	return &APITokenModel{
		ID:          e.ID,
		UserID:      e.UserID,
		HashedToken: e.HashedToken,
		CreatedAt:   e.CreatedAt,
	}
}
