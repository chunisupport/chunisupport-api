package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

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

func (r *goalRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) (*entity.Goal, error) {
	var m models.GoalModel
	query := `SELECT id, user_id, title, achievement_type_id, achievement_params, attributes, invert, created_at FROM goals WHERE id = ? AND user_id = ?`
	if err := exec.GetContext(ctx, &m, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrGoalNotFound, err)
		}
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
	if id < 0 || id > math.MaxUint32 {
		return fmt.Errorf("goals.id out of range: %d", id)
	}
	goal.ID = uint32(id)
	return nil
}

func (r *goalRepository) Update(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	query := `UPDATE goals SET title = ?, achievement_type_id = ?, achievement_params = ?, attributes = ?, invert = ? WHERE id = ? AND user_id = ?`
	res, err := exec.ExecContext(ctx, query, goal.Title, goal.AchievementTypeID, goal.AchievementParams, goal.Attributes, goal.Invert, goal.ID, goal.UserID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrGoalNotFound
	}
	return nil
}

func (r *goalRepository) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) error {
	query := `DELETE FROM goals WHERE id = ? AND user_id = ?`
	res, err := exec.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrGoalNotFound
	}
	return nil
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

func (r *goalRepository) GetTargetStats(ctx context.Context, exec repository.Executor, filter repository.GoalTargetFilter) (*repository.GoalTargetStats, error) {
	where := []string{"s.is_deleted = 0"}
	args := make([]any, 0, 8)

	if len(filter.DifficultyIDs) > 0 {
		where = append(where, "c.difficulty_id IN (?)")
		args = append(args, filter.DifficultyIDs)
	}
	if len(filter.GenreIDs) > 0 {
		where = append(where, "s.genre_id IN (?)")
		args = append(args, filter.GenreIDs)
	}
	if len(filter.VersionRanges) > 0 {
		versionWhere := make([]string, 0, len(filter.VersionRanges))
		for _, versionRange := range filter.VersionRanges {
			if versionRange.To == nil {
				versionWhere = append(versionWhere, "s.released_at >= ?")
				args = append(args, versionRange.From)
				continue
			}
			versionWhere = append(versionWhere, "(s.released_at >= ? AND s.released_at < ?)")
			args = append(args, versionRange.From, *versionRange.To)
		}
		where = append(where, "("+strings.Join(versionWhere, " OR ")+")")
	}
	if filter.ConstMin != nil {
		where = append(where, "c.const >= ?")
		args = append(args, *filter.ConstMin)
	}
	if filter.ConstMax != nil {
		where = append(where, "c.const <= ?")
		args = append(args, *filter.ConstMax)
	}

	query := `
		SELECT
			COUNT(*) AS chart_count,
			COALESCE(SUM(c.const), 0) AS total_chart_const
		FROM charts c
		INNER JOIN songs s ON s.id = c.song_id
		WHERE ` + strings.Join(where, " AND ")
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var row struct {
		ChartCount      int     `db:"chart_count"`
		TotalChartConst float64 `db:"total_chart_const"`
	}
	if err := exec.GetContext(ctx, &row, query, args...); err != nil {
		return nil, err
	}

	return &repository.GoalTargetStats{ChartCount: row.ChartCount, TotalChartConst: row.TotalChartConst}, nil
}
