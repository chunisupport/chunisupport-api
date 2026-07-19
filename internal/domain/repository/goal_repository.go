package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalRepository は目標の永続化を扱います。
type GoalRepository interface {
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.Goal, error)
	// FindByIDAndUserID は対象が存在しない場合に ErrGoalNotFound を返します。
	FindByIDAndUserID(ctx context.Context, exec Executor, id uint32, userID int) (*entity.Goal, error)
	Create(ctx context.Context, exec Executor, goal *entity.Goal) error
	// Save は集約の現在状態を保存し、対象が存在しない場合に ErrGoalNotFound を返します。
	Save(ctx context.Context, exec Executor, goal *entity.Goal) error
	// DeleteByIDAndUserID は対象が存在しない場合に ErrGoalNotFound を返します。
	DeleteByIDAndUserID(ctx context.Context, exec Executor, id uint32, userID int) error
	// SaveGoalArrangement はユーザーの全目標についてグループ所属とグループ内表示順を保存します。
	SaveGoalArrangement(ctx context.Context, exec Executor, arrangement *entity.GoalArrangement) error
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	CountByUserIDAndGroupID(ctx context.Context, exec Executor, userID int, groupID *uint32) (int, error)
	LockUserByID(ctx context.Context, exec Executor, userID int) error
	GetTargetStats(ctx context.Context, exec Executor, filter GoalTargetFilter) (*GoalTargetStats, error)
}

// GoalTargetFilter は目標対象譜面の絞り込み条件です。
type GoalTargetFilter struct {
	DifficultyIDs []int
	GenreIDs      []int
	VersionRanges []VersionRange
	ConstMin      *float64
	ConstMax      *float64
	OPTargetOnly  bool
}

type VersionRange struct {
	From time.Time
	To   *time.Time
}

// GoalTargetStats は絞り込み結果から得られる上限計算用統計です。
type GoalTargetStats struct {
	ChartCount      int
	SongCount       int
	TotalChartConst float64
}
