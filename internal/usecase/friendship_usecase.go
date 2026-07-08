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
	AcceptRequest(ctx context.Context, userID int, requesterID int) error
	RejectRequest(ctx context.Context, userID int, requesterID int) error
	Remove(ctx context.Context, userID int, friendUserID int) error
}

// FriendshipUserOutput はフレンド・申請一覧の相手ユーザー概要です。
type FriendshipUserOutput struct {
	UserID      int
	Username    string
	PlayerLevel *int
	PlayerName  *string
	Rating      *float64
	RequestedAt time.Time
	AcceptedAt  *time.Time
}
