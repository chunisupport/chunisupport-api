package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerRecordHistory は通常譜面の退避対象を表します。
type PlayerRecordHistory struct {
	PlayerID int
	ChartID  int
	State    PlayerRecordState
}

// PlayerWorldsendRecordHistory はWORLD'S END譜面の退避対象を表します。
type PlayerWorldsendRecordHistory struct {
	PlayerID         int
	WorldsendChartID int
	State            WorldsendRecordState
}

// PlayerCourseRecordHistory はコース記録の退避対象を表します。
type PlayerCourseRecordHistory struct {
	PlayerID int
	CourseID int
	State    CourseRecordState
}

// ScoreHistoryRepository はスコア履歴の保存とタイムライン取得を扱います。
type ScoreHistoryRepository interface {
	BulkInsertStandard(ctx context.Context, exec Executor, rows []PlayerRecordHistory) error
	BulkInsertWorldsend(ctx context.Context, exec Executor, rows []PlayerWorldsendRecordHistory) error
	BulkInsertCourse(ctx context.Context, exec Executor, rows []PlayerCourseRecordHistory) error
	PruneStandardOverLimit(ctx context.Context, exec Executor, playerID int, chartIDs []int) error
	PruneWorldsendOverLimit(ctx context.Context, exec Executor, playerID int, chartIDs []int) error
	PruneCourseOverLimit(ctx context.Context, exec Executor, playerID int, courseIDs []int) error
	FindStandardTimeline(ctx context.Context, playerID, chartID int) ([]entity.ScoreHistoryEntry, error)
	FindWorldsendTimeline(ctx context.Context, playerID, worldsendChartID int) ([]entity.ScoreHistoryEntry, error)
	FindCourseTimeline(ctx context.Context, playerID, courseID int) ([]entity.CourseScoreHistoryEntry, error)
}
