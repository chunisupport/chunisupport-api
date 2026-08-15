package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalGroupRepository は目標グループ集約の永続化を扱います。
type GoalGroupRepository interface {
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.GoalGroup, error)
	FindByIDAndUserID(ctx context.Context, exec Executor, id uint32, userID int) (*entity.GoalGroup, error)
	// Save はIDが0なら作成し、それ以外は集約の現在状態を保存します。
	Save(ctx context.Context, exec Executor, group *entity.GoalGroup) error
	DeleteByIDAndUserID(ctx context.Context, exec Executor, id uint32, userID int) error
	SaveOrder(ctx context.Context, exec Executor, order *entity.GoalGroupOrder) error
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
}
