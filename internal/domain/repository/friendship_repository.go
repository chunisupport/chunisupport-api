package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// FriendshipUserSummary はフレンド・申請一覧で返す相手ユーザーの概要です。
type FriendshipUserSummary struct {
	UserID      int
	Username    string
	PlayerLevel *int
	PlayerName  *string
	Rating      *float64
}

// FriendshipWithUserSummary は片方向フレンド関係と相手ユーザー概要をまとめた読み取りモデルです。
type FriendshipWithUserSummary struct {
	Friendship *entity.Friendship
	User       *FriendshipUserSummary
}

// FriendshipRepository はフレンド関係の永続化を扱います。
type FriendshipRepository interface {
	Find(ctx context.Context, exec Executor, userID int, friendUserID int) (*entity.Friendship, error)
	Save(ctx context.Context, exec Executor, friendship *entity.Friendship) error
	Delete(ctx context.Context, exec Executor, userID int, friendUserID int) error
	DeletePending(ctx context.Context, exec Executor, userID int, friendUserID int) error
	DeletePair(ctx context.Context, exec Executor, userID int, friendUserID int) error
	CountOutgoingActive(ctx context.Context, exec Executor, userID int) (int, error)
	ListFriends(ctx context.Context, exec Executor, userID int) ([]*FriendshipWithUserSummary, error)
	ListReceivedRequests(ctx context.Context, exec Executor, userID int) ([]*FriendshipWithUserSummary, error)
	ListSentRequests(ctx context.Context, exec Executor, userID int) ([]*FriendshipWithUserSummary, error)
	ExistsMutualAccepted(ctx context.Context, exec Executor, userID int, friendUserID int) (bool, error)
}
