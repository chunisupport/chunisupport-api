package repository

import (
	"context"
	"time"
)

// CourseRankingRecord はプレイヤーのコース自己ベストです。
type CourseRankingRecord struct {
	UserID     int
	Username   string
	PlayerName string
	Score      uint32
	IsClear    bool
	ComboLamp  string
	UpdatedAt  time.Time
}

// CourseRankingQueryService は閲覧範囲に応じたコース記録をスコア降順で返します。
type CourseRankingQueryService interface {
	ListFriends(ctx context.Context, exec Executor, courseID, userID int) ([]CourseRankingRecord, error)
	ListTop(ctx context.Context, exec Executor, courseID, limit int) ([]CourseRankingRecord, int, error)
}
