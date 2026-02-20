package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type goalRepository struct {
	db *sqlx.DB
}

// NewGoalRepository は新しいGoalRepositoryを生成します。
func NewGoalRepository(db *sqlx.DB) repository.GoalRepository {
	return &goalRepository{db: db}
}

func (r *goalRepository) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	var goalModels []*models.GoalModel
	query := `SELECT id, user_id, title, achievement_type_id, achievement_params, attributes, invert, created_at FROM goals WHERE user_id = ? ORDER BY created_at ASC, id ASC`
	if err := exec.SelectContext(ctx, &goalModels, query, userID); err != nil {
		return nil, err
	}
	goals := make([]*entity.Goal, 0, len(goalModels))
	for _, m := range goalModels {
		goals = append(goals, m.ToEntity())
	}
	return goals, nil
}

func (r *goalRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id int64, userID int) (*entity.Goal, error) {
	var m models.GoalModel
	query := `SELECT id, user_id, title, achievement_type_id, achievement_params, attributes, invert, created_at FROM goals WHERE id = ? AND user_id = ?`
	if err := exec.GetContext(ctx, &m, query, id, userID); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *goalRepository) Create(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	query := `INSERT INTO goals (user_id, title, achievement_type_id, achievement_params, attributes, invert) VALUES (?, ?, ?, ?, ?, ?)`
	res, err := exec.ExecContext(ctx, query, goal.UserID, goal.Title, goal.AchievementTypeID, goal.AchievementParams, goal.Attributes, goal.Invert)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	goal.ID = id
	return nil
}

func (r *goalRepository) Update(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	query := `UPDATE goals SET title = ?, achievement_type_id = ?, achievement_params = ?, attributes = ?, invert = ? WHERE id = ? AND user_id = ?`
	_, err := exec.ExecContext(ctx, query, goal.Title, goal.AchievementTypeID, goal.AchievementParams, goal.Attributes, goal.Invert, goal.ID, goal.UserID)
	return err
}

func (r *goalRepository) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id int64, userID int) error {
	query := `DELETE FROM goals WHERE id = ? AND user_id = ?`
	_, err := exec.ExecContext(ctx, query, id, userID)
	return err
}

func (r *goalRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	if err := exec.GetContext(ctx, &count, `SELECT COUNT(*) FROM goals WHERE user_id = ?`, userID); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *goalRepository) LockUserByID(ctx context.Context, exec repository.Executor, userID int) error {
	var id int
	return exec.GetContext(ctx, &id, `SELECT id FROM users WHERE id = ? FOR UPDATE`, userID)
}
