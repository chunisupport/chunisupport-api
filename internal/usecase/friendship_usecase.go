package usecase

import (
	"context"
	"time"
)

// FriendshipUsecase はフレンド機能のユースケースを定義します。
type FriendshipUsecase interface {
	SendRequest(ctx context.Context, userID int, username string) error
	ListFriends(ctx context.Context, userID int) ([]*FriendshipUserOutput, error)
	ListReceivedRequests(ctx context.Context, userID int) ([]*FriendshipUserOutput, error)
	ListSentRequests(ctx context.Context, userID int) ([]*FriendshipUserOutput, error)
	AcceptRequest(ctx context.Context, userID int, requesterUsername string) error
	RejectRequest(ctx context.Context, userID int, requesterUsername string) error
	CancelRequest(ctx context.Context, userID int, targetUsername string) error
	Remove(ctx context.Context, userID int, friendUsername string) error
	GetSlots(ctx context.Context, userID int) (*FriendshipSlotsOutput, error)
}

// FriendshipSlotsOutput はフレンド枠の上限と使用数です。
// 使用数は自分から外向きの pending / accepted の合計で、申請・承認時の上限判定と同じ数え方です。
type FriendshipSlotsOutput struct {
	Limit int
	Used  int
}

// FriendshipUserOutput はフレンド・申請一覧の相手ユーザー概要です。
type FriendshipUserOutput struct {
	Username    string
	PlayerLevel *int
	PlayerName  *string
	Rating      *float64
	// OverpowerValue はOVER POWER値です。
	OverpowerValue *float64
	// PossessionID はポゼッションマスタのIDです。
	PossessionID *int
	IsPrivate    bool
	RequestedAt  time.Time
	AcceptedAt   *time.Time
}
