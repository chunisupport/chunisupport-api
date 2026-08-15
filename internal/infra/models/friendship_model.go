package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// FriendshipModel はDB用の片方向フレンド関係モデルです。
type FriendshipModel struct {
	UserID       int        `db:"user_id"`
	FriendUserID int        `db:"friend_user_id"`
	StatusID     int        `db:"status_id"`
	RequestedAt  time.Time  `db:"requested_at"`
	AcceptedAt   *time.Time `db:"accepted_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

func (m *FriendshipModel) ToEntity() *entity.Friendship {
	return &entity.Friendship{
		UserID:       m.UserID,
		FriendUserID: m.FriendUserID,
		StatusID:     m.StatusID,
		RequestedAt:  m.RequestedAt,
		AcceptedAt:   m.AcceptedAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func FromFriendshipEntity(e *entity.Friendship) *FriendshipModel {
	return &FriendshipModel{
		UserID:       e.UserID,
		FriendUserID: e.FriendUserID,
		StatusID:     e.StatusID,
		RequestedAt:  e.RequestedAt,
		AcceptedAt:   e.AcceptedAt,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
