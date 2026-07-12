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
	FindByOfficialIdx(ctx context.Context, exec Executor, officialIdx string, includeDeleted bool) (*entity.Course, error)
	FindByOfficialIdxList(ctx context.Context, exec Executor, officialIdxList []string) (map[string]*entity.Course, error)
	FindClassByName(ctx context.Context, exec Executor, name string) (*entity.CourseClass, error)
	Create(ctx context.Context, exec Executor, course *entity.Course) error
	Save(ctx context.Context, exec Executor, course *entity.Course) error
	FindRecordsByPlayerID(ctx context.Context, exec Executor, playerID int, includeDeleted, includeNoPlay bool) ([]*entity.PlayerCourseRecord, error)
	FindRecordStatesByCourseIDs(ctx context.Context, exec Executor, playerID int, courseIDs []int) (map[int]CourseRecordState, error)
	UpsertRecords(ctx context.Context, exec Executor, records []CourseRecordForUpsert) error
}
