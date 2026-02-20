package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalRepository は目標の永続化を扱います。
type GoalRepository interface {
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.Goal, error)
	FindByIDAndUserID(ctx context.Context, exec Executor, id uint32, userID int) (*entity.Goal, error)
	Create(ctx context.Context, exec Executor, goal *entity.Goal) error
	Update(ctx context.Context, exec Executor, goal *entity.Goal) error
	DeleteByIDAndUserID(ctx context.Context, exec Executor, id uint32, userID int) error
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	LockUserByID(ctx context.Context, exec Executor, userID int) error
	GetTargetStats(ctx context.Context, exec Executor, filter GoalTargetFilter) (*GoalTargetStats, error)
}

// GoalTargetFilter は目標対象譜面の絞り込み条件です。
type GoalTargetFilter struct {
	DifficultyID          *int
	GenreID               *int
	VersionReleasedAt     *time.Time
	VersionReleasedBefore *time.Time
	ConstMin              *float64
	ConstMax              *float64
}

// GoalTargetStats は絞り込み結果から得られる上限計算用統計です。
type GoalTargetStats struct {
	ChartCount      int
	TotalChartConst float64
}
