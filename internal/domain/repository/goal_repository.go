package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalRepository は目標の永続化を扱います。
type GoalRepository interface {
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.Goal, error)
	FindByIDAndUserID(ctx context.Context, exec Executor, id int64, userID int) (*entity.Goal, error)
	Create(ctx context.Context, exec Executor, goal *entity.Goal) error
	Update(ctx context.Context, exec Executor, goal *entity.Goal) error
	DeleteByIDAndUserID(ctx context.Context, exec Executor, id int64, userID int) error
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	LockUserByID(ctx context.Context, exec Executor, userID int) error
}
