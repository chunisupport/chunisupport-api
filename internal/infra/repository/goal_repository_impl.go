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
	"github.com/chunisupport/chunisupport-api/internal/info"
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
	query := `SELECT g.id, g.user_id, g.group_id, g.title, g.achievement_type_id, g.achievement_params, g.attributes, g.invert, g.sort_order, g.created_at
		FROM goals g
		LEFT JOIN goal_groups gg ON gg.id = g.group_id AND gg.user_id = g.user_id
		WHERE g.user_id = ?
		ORDER BY (g.group_id IS NULL) ASC, gg.sort_order ASC, g.sort_order ASC, g.id ASC`
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
	query := `SELECT id, user_id, group_id, title, achievement_type_id, achievement_params, attributes, invert, sort_order, created_at FROM goals WHERE id = ? AND user_id = ?`
	if err := exec.GetContext(ctx, &m, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrGoalNotFound, err)
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *goalRepository) Create(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	query := `INSERT INTO goals (user_id, group_id, title, achievement_type_id, achievement_params, attributes, invert, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := exec.ExecContext(ctx, query, goal.UserID, goal.GroupID, goal.Title, goal.AchievementTypeID, goal.AchievementParams, goal.Attributes, goal.Invert, goal.SortOrder)
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

func (r *goalRepository) Save(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	query := `UPDATE goals SET group_id = ?, title = ?, achievement_type_id = ?, achievement_params = ?, attributes = ?, invert = ?, sort_order = ? WHERE id = ? AND user_id = ?`
	res, err := exec.ExecContext(ctx, query, goal.GroupID, goal.Title, goal.AchievementTypeID, goal.AchievementParams, goal.Attributes, goal.Invert, goal.SortOrder, goal.ID, goal.UserID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var exists int
		if err := exec.GetContext(ctx, &exists, `SELECT COUNT(*) FROM goals WHERE id = ? AND user_id = ?`, goal.ID, goal.UserID); err != nil {
			return err
		}
		if exists == 0 {
			return repository.ErrGoalNotFound
		}
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

// SaveGoalArrangement は最大100件の全状態を単一UPDATEで保存し、グループ間移動も原子的に反映します。
func (r *goalRepository) SaveGoalArrangement(ctx context.Context, exec repository.Executor, arrangement *entity.GoalArrangement) error {
	goals := arrangement.Goals()
	if err := validateGoalArrangementPersistence(ctx, exec, arrangement.UserID(), goals); err != nil {
		return err
	}
	if len(goals) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`UPDATE goals SET group_id = CASE id`)
	args := make([]any, 0, len(goals)*5+1)
	for _, goal := range goals {
		query.WriteString(` WHEN ? THEN ?`)
		args = append(args, goal.ID, goal.GroupID)
	}
	query.WriteString(` ELSE group_id END, sort_order = CASE id`)
	for _, goal := range goals {
		query.WriteString(` WHEN ? THEN ?`)
		args = append(args, goal.ID, goal.SortOrder)
	}
	query.WriteString(` ELSE sort_order END WHERE user_id = ?`)
	args = append(args, arrangement.UserID())

	_, err := exec.ExecContext(ctx, query.String(), args...)
	return err
}

func validateGoalArrangementPersistence(ctx context.Context, exec repository.Executor, userID int, goals []*entity.Goal) error {
	var total int
	if err := exec.GetContext(ctx, &total, `SELECT COUNT(*) FROM goals WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if total != len(goals) {
		return repository.ErrGoalOrderInconsistent
	}
	if len(goals) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`SELECT COUNT(*) FROM goals WHERE user_id = ? AND id IN (`)
	args := make([]any, 0, len(goals)+1)
	args = append(args, userID)
	for i, goal := range goals {
		if i > 0 {
			query.WriteString(`, `)
		}
		query.WriteString(`?`)
		args = append(args, goal.ID)
	}
	query.WriteString(`)`)

	var matched int
	if err := exec.GetContext(ctx, &matched, query.String(), args...); err != nil {
		return err
	}
	if matched != len(goals) {
		return repository.ErrGoalOrderInconsistent
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

func (r *goalRepository) CountByUserIDAndGroupID(ctx context.Context, exec repository.Executor, userID int, groupID *uint32) (int, error) {
	var count int
	if groupID == nil {
		if err := exec.GetContext(ctx, &count, `SELECT COUNT(*) FROM goals WHERE user_id = ? AND group_id IS NULL`, userID); err != nil {
			return 0, err
		}
		return count, nil
	}
	if err := exec.GetContext(ctx, &count, `SELECT COUNT(*) FROM goals WHERE user_id = ? AND group_id = ?`, userID, *groupID); err != nil {
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
	if filter.OPTargetOnly {
		where = append(where, `NOT EXISTS (
			SELECT 1
			FROM charts higher
			WHERE higher.song_id = c.song_id
			  AND (higher.const > c.const OR (higher.const = c.const AND higher.difficulty_id > c.difficulty_id))
		)`)
	}

	query := `
		SELECT
			COUNT(*) AS chart_count,
			COUNT(DISTINCT rainbow.song_id) AS song_count,
			COALESCE(SUM(c.const), 0) AS total_chart_const
		FROM charts c
		INNER JOIN songs s ON s.id = c.song_id
		LEFT JOIN (
			SELECT song_id
			FROM charts
			WHERE difficulty_id BETWEEN %d AND %d
			GROUP BY song_id
			HAVING COUNT(DISTINCT difficulty_id) = %d
		) rainbow ON rainbow.song_id = s.id
		WHERE ` + strings.Join(where, " AND ")
	query = fmt.Sprintf(
		query,
		info.RainbowRequiredDifficultyMinID,
		info.RainbowRequiredDifficultyMaxID,
		info.RainbowRequiredDifficultyCount,
	)
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var row struct {
		ChartCount      int     `db:"chart_count"`
		SongCount       int     `db:"song_count"`
		TotalChartConst float64 `db:"total_chart_const"`
	}
	if err := exec.GetContext(ctx, &row, query, args...); err != nil {
		return nil, err
	}

	return &repository.GoalTargetStats{
		ChartCount:      row.ChartCount,
		SongCount:       row.SongCount,
		TotalChartConst: row.TotalChartConst,
	}, nil
}
