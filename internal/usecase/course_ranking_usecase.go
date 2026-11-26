package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// CourseRankingEntry は順位付きのコース自己ベストです。
type CourseRankingEntry struct {
	Rank       int
	Username   string
	PlayerName string
	Score      uint32
	IsClear    bool
	ComboLamp  *string
	UpdatedAt  time.Time
	IsSelf     bool
}

// CourseRankingResult はコース名、ランキングと対象者数を返します。
type CourseRankingResult struct {
	CourseID string
	Name     string
	Class    string
	Ranking  []CourseRankingEntry
	Total    int
}

// CourseRankingUsecase はコース別にフレンド用と管理者用のランキングを取得します。
type CourseRankingUsecase interface {
	GetFriends(ctx context.Context, userID int, displayID string) (*CourseRankingResult, error)
	GetAdmin(ctx context.Context, displayID string) (*CourseRankingResult, error)
}

type courseRankingUsecase struct {
	exec    repository.Executor
	courses repository.CourseRepository
	records repository.CourseRankingQueryService
}

// NewCourseRankingUsecase はコースマスタと記録の参照を組み合わせます。
func NewCourseRankingUsecase(exec repository.Executor, courses repository.CourseRepository, records repository.CourseRankingQueryService) CourseRankingUsecase {
	return &courseRankingUsecase{exec: exec, courses: courses, records: records}
}

func (u *courseRankingUsecase) GetFriends(ctx context.Context, userID int, displayID string) (*CourseRankingResult, error) {
	course, err := u.courses.FindByDisplayID(ctx, u.exec, displayID, false)
	if err != nil {
		return nil, err
	}
	rows, err := u.records.ListFriends(ctx, u.exec, course.ID, userID)
	if err != nil {
		return nil, err
	}
	return &CourseRankingResult{
		CourseID: displayID, Name: course.Name, Class: course.CourseClass.Name,
		Ranking: rankCourseRecords(rows, userID), Total: len(rows),
	}, nil
}

func (u *courseRankingUsecase) GetAdmin(ctx context.Context, displayID string) (*CourseRankingResult, error) {
	course, err := u.courses.FindByDisplayID(ctx, u.exec, displayID, false)
	if err != nil {
		return nil, err
	}
	rows, total, err := u.records.ListTop(ctx, u.exec, course.ID, info.AdminRankingLimit)
	if err != nil {
		return nil, err
	}
	return &CourseRankingResult{
		CourseID: displayID, Name: course.Name, Class: course.CourseClass.Name,
		Ranking: rankCourseRecords(rows, 0), Total: total,
	}, nil
}

func rankCourseRecords(rows []repository.CourseRankingRecord, userID int) []CourseRankingEntry {
	entries := make([]CourseRankingEntry, 0, len(rows))
	var ranker scoreRanker
	for _, row := range rows {
		entries = append(entries, CourseRankingEntry{
			Rank: ranker.Next(row.Score), Username: row.Username, PlayerName: row.PlayerName,
			Score: row.Score, IsClear: row.IsClear, ComboLamp: rankingLampNamePtr(row.ComboLamp),
			UpdatedAt: row.UpdatedAt, IsSelf: row.UserID == userID,
		})
	}
	return entries
}
