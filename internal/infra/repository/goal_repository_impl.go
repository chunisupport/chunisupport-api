package repository

import (
	"context"
	"database/sql"

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

func (r *goalRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	var rows []models.GoalModel
	query := `SELECT id, user_id, title, achievement_type_id, achievement_params, attributes, invert, created_at FROM goals WHERE user_id = ? ORDER BY created_at ASC, id ASC`
	if err := exec.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, err
	}
	goals := make([]*entity.Goal, 0, len(rows))
	for i := range rows {
		goal, err := rows[i].ToEntity()
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}
	return goals, nil
}

func (r *goalRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id int, userID int) (*entity.Goal, error) {
	var row models.GoalModel
	query := `SELECT id, user_id, title, achievement_type_id, achievement_params, attributes, invert, created_at FROM goals WHERE id = ? AND user_id = ?`
	if err := exec.GetContext(ctx, &row, query, id, userID); err != nil {
		return nil, err
	}
	return row.ToEntity()
}

func (r *goalRepository) Create(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	model, err := models.FromGoalEntity(goal)
	if err != nil {
		return err
	}
	query := `INSERT INTO goals (user_id, title, achievement_type_id, achievement_params, attributes, invert) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := exec.ExecContext(ctx, query, model.UserID, model.Title, model.AchievementTypeID, model.AchievementParams, model.Attributes, model.Invert)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	goal.ID = int(id)
	return nil
}

func (r *goalRepository) Update(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	model, err := models.FromGoalEntity(goal)
	if err != nil {
		return err
	}
	query := `UPDATE goals SET title = ?, achievement_type_id = ?, achievement_params = ?, attributes = ?, invert = ? WHERE id = ? AND user_id = ?`
	result, err := exec.ExecContext(ctx, query, model.Title, model.AchievementTypeID, model.AchievementParams, model.Attributes, model.Invert, model.ID, model.UserID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *goalRepository) Delete(ctx context.Context, exec repository.Executor, id int, userID int) error {
	query := `DELETE FROM goals WHERE id = ? AND user_id = ?`
	result, err := exec.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *goalRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM goals WHERE user_id = ?`
	if err := exec.GetContext(ctx, &count, query, userID); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *goalRepository) LockUser(ctx context.Context, exec repository.Executor, userID int) error {
	var id int
	query := `SELECT id FROM users WHERE id = ? FOR UPDATE`
	if err := exec.GetContext(ctx, &id, query, userID); err != nil {
		return err
	}
	return nil
}
