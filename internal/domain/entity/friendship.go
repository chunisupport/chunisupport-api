package entity

import (
	"errors"
	"time"
)

const (
	FriendshipStatusPending  = 1
	FriendshipStatusAccepted = 2
	FriendshipStatusBlocked  = 3
)

// Friendship はユーザー間の片方向フレンド関係を表します。
type Friendship struct {
	UserID       int
	FriendUserID int
	StatusID     int
	RequestedAt  time.Time
	AcceptedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewFriendRequest は片方向のフレンド申請を生成します。
func NewFriendRequest(userID int, friendUserID int, now time.Time) (*Friendship, error) {
	friendship := &Friendship{
		UserID:       userID,
		FriendUserID: friendUserID,
		StatusID:     FriendshipStatusPending,
		RequestedAt:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := friendship.Validate(); err != nil {
		return nil, err
	}
	return friendship, nil
}

// NewAcceptedFriendship は承認済みの片方向フレンド関係を生成します。
func NewAcceptedFriendship(userID int, friendUserID int, requestedAt time.Time, acceptedAt time.Time) (*Friendship, error) {
	friendship := &Friendship{
		UserID:       userID,
		FriendUserID: friendUserID,
		StatusID:     FriendshipStatusAccepted,
		RequestedAt:  requestedAt,
		AcceptedAt:   &acceptedAt,
		CreatedAt:    requestedAt,
		UpdatedAt:    acceptedAt,
	}
	if err := friendship.Validate(); err != nil {
		return nil, err
	}
	return friendship, nil
}

// Accept は申請中のフレンド関係を承認済みに変更します。
func (f *Friendship) Accept(acceptedAt time.Time) error {
	if f == nil {
		return errors.New("friendship is nil")
	}
	if f.StatusID != FriendshipStatusPending && f.StatusID != FriendshipStatusAccepted {
		return errors.New("friendship status cannot be accepted")
	}
	f.StatusID = FriendshipStatusAccepted
	f.AcceptedAt = &acceptedAt
	f.UpdatedAt = acceptedAt
	return f.Validate()
}

// IsPending は申請中かを判定します。
func (f *Friendship) IsPending() bool {
	return f != nil && f.StatusID == FriendshipStatusPending
}

// IsAccepted は承認済みかを判定します。
func (f *Friendship) IsAccepted() bool {
	return f != nil && f.StatusID == FriendshipStatusAccepted
}

// Validate はフレンド関係の不変条件を検証します。
func (f *Friendship) Validate() error {
	if f == nil {
		return errors.New("friendship is nil")
	}
	if f.UserID <= 0 {
		return errors.New("user id must be positive")
	}
	if f.FriendUserID <= 0 {
		return errors.New("friend user id must be positive")
	}
	if f.UserID == f.FriendUserID {
		return errors.New("self friendship is not allowed")
	}
	switch f.StatusID {
	case FriendshipStatusPending:
		if f.AcceptedAt != nil {
			return errors.New("pending friendship must not have accepted at")
		}
	case FriendshipStatusAccepted:
		if f.AcceptedAt == nil {
			return errors.New("accepted friendship must have accepted at")
		}
	case FriendshipStatusBlocked:
	default:
		return errors.New("invalid friendship status")
	}
	if f.RequestedAt.IsZero() {
		return errors.New("requested at is required")
	}
	return nil
}
