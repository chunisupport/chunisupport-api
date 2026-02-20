package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalRepository は目標データへのアクセスを抽象化するインターフェースです。
type GoalRepository interface {
	FindByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.Goal, error)
	FindByIDAndUserID(ctx context.Context, exec Executor, id int, userID int) (*entity.Goal, error)
	Create(ctx context.Context, exec Executor, goal *entity.Goal) error
	Update(ctx context.Context, exec Executor, goal *entity.Goal) error
	Delete(ctx context.Context, exec Executor, id int, userID int) error
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	LockUser(ctx context.Context, exec Executor, userID int) error
}
