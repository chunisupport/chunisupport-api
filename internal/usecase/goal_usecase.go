package usecase

import "context"

// GoalUsecase は目標機能のユースケースです。
type GoalUsecase interface {
	List(ctx context.Context, userID int) ([]*GoalOutput, error)
	Create(ctx context.Context, userID int, input *GoalInput) (*GoalOutput, error)
	Update(ctx context.Context, userID int, id int64, input *GoalInput) (*GoalOutput, error)
	Delete(ctx context.Context, userID int, id int64) error
}

// GoalInput は目標の作成・更新入力です。
type GoalInput struct {
	Title             string
	AchievementType   string
	AchievementParams []byte
	Attributes        []byte
	Invert            bool
}

// GoalOutput は目標API向けの出力です。
type GoalOutput struct {
	ID                int64
	Title             string
	AchievementType   string
	AchievementParams map[string]any
	Attributes        map[string]any
	Invert            bool
	CreatedAt         string
}
