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
}

// FriendshipUserOutput はフレンド・申請一覧の相手ユーザー概要です。
type FriendshipUserOutput struct {
	Username    string
	PlayerLevel *int
	PlayerName  *string
	Rating      *float64
	IsPrivate   bool
	RequestedAt time.Time
	AcceptedAt  *time.Time
}
