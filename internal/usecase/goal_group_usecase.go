package usecase

import (
	"context"
	"time"
)

// GoalGroupUsecase は目標グループの操作を提供します。
type GoalGroupUsecase interface {
	List(ctx context.Context, userID int) ([]*GoalGroupOutput, error)
	Create(ctx context.Context, userID int, name string) (*GoalGroupOutput, error)
	Update(ctx context.Context, userID int, id uint32, name string) (*GoalGroupOutput, error)
	Delete(ctx context.Context, userID int, id uint32) error
	Reorder(ctx context.Context, userID int, orderedGroupIDs []uint32) error
}

// GoalGroupOutput は目標グループAPI向けの出力です。
type GoalGroupOutput struct {
	ID        uint32
	Name      string
	SortOrder uint16
	CreatedAt time.Time
}
