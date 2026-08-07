package usecase

import (
	"context"
	"time"
)

// GoalUsecase は目標機能のユースケースです。
type GoalUsecase interface {
	List(ctx context.Context, userID int) ([]*GoalOutput, error)
	Create(ctx context.Context, userID int, input *GoalInput) (*GoalOutput, error)
	Update(ctx context.Context, userID int, id uint32, input *GoalInput) (*GoalOutput, error)
	Delete(ctx context.Context, userID int, id uint32) error
	Reorder(ctx context.Context, userID int, groupID *uint32, orderedGoalIDs []uint32) error
}

// GoalInput は目標の作成・更新入力です。
type GoalInput struct {
	GroupID           *uint32
	Title             string
	AchievementType   string
	AchievementParams []byte
	Attributes        []byte
	InvertValue       bool
	InvertPercentage  bool
}

// GoalOutput は目標API向けの出力です。
type GoalOutput struct {
	ID                uint32
	GroupID           *uint32
	Title             string
	AchievementType   string
	AchievementParams map[string]any
	Attributes        map[string]any
	InvertValue       bool
	InvertPercentage  bool
	SortOrder         uint16
	CreatedAt         time.Time
}
