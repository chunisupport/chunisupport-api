package models

import (
	"fmt"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/google/uuid"
)

// SessionModel はデータベース用のSessionモデルです。
type SessionModel struct {
	ID        []byte    `db:"id"` // BINARY(16) でUUIDを保存
	UserID    int       `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// ToEntity はSessionModelをentity.Sessionに変換します。
// UUIDのパースに失敗した場合はエラーを返します。
func (m *SessionModel) ToEntity() (*entity.Session, error) {
	if len(m.ID) != 16 {
		return nil, fmt.Errorf("invalid session ID length: expected 16 bytes, got %d", len(m.ID))
	}

	var uuidObj uuid.UUID
	copy(uuidObj[:], m.ID)

	return &entity.Session{
		ID:        uuidObj.String(),
		UserID:    m.UserID,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}, nil
}

// FromSessionEntity はentity.SessionをSessionModelに変換します。
// UUIDのパースに失敗した場合はエラーを返します。
func FromSessionEntity(e *entity.Session) (*SessionModel, error) {
	uuidObj, err := uuid.Parse(e.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %w", err)
	}

	return &SessionModel{
		ID:        uuidObj[:],
		UserID:    e.UserID,
		ExpiresAt: e.ExpiresAt,
		CreatedAt: e.CreatedAt,
	}, nil
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
