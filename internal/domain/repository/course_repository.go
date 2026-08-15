package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

type CourseRecordState struct {
	Score       int
	IsClear     bool
	ComboLampID int
	UpdatedAt   time.Time
}

type CourseRecordForUpsert struct {
	PlayerID int
	CourseID int
	State    CourseRecordState
}

// CourseRepository はコース集約とプレイヤーのコース記録を永続化します。
type CourseRepository interface {
	FindAll(ctx context.Context, exec Executor, includeDeleted bool) ([]*entity.Course, error)
	FindByDisplayID(ctx context.Context, exec Executor, displayID string, includeDeleted bool) (*entity.Course, error)
	FindByOfficialIdx(ctx context.Context, exec Executor, officialIdx string, includeDeleted bool) (*entity.Course, error)
	FindByOfficialIdxList(ctx context.Context, exec Executor, officialIdxList []string) (map[string]*entity.Course, error)
	FindClassByName(ctx context.Context, exec Executor, name string) (*entity.CourseClass, error)
	// FindLatestUpdatedAt は courses.updated_at の最大値を返します。
	// コースが0件の場合は nil を返します。
	FindLatestUpdatedAt(ctx context.Context, exec Executor) (*time.Time, error)
	Create(ctx context.Context, exec Executor, course *entity.Course) error
	Save(ctx context.Context, exec Executor, course *entity.Course) error
	FindRecordsByPlayerID(ctx context.Context, exec Executor, playerID int, includeDeleted, includeNoPlay bool) ([]*entity.PlayerCourseRecord, error)
	FindRecordStatesByCourseIDs(ctx context.Context, exec Executor, playerID int, courseIDs []int) (map[int]CourseRecordState, error)
	UpsertRecords(ctx context.Context, exec Executor, records []CourseRecordForUpsert) error
}
