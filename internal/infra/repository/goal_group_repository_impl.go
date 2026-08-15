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
)

type goalGroupRepository struct{}

// NewGoalGroupRepository は目標グループリポジトリを生成します。
func NewGoalGroupRepository() repository.GoalGroupRepository {
	return &goalGroupRepository{}
}

func (r *goalGroupRepository) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.GoalGroup, error) {
	var rows []*models.GoalGroupModel
	if err := exec.SelectContext(ctx, &rows, `SELECT id, user_id, name, sort_order, created_at FROM goal_groups WHERE user_id = ? ORDER BY sort_order ASC, id ASC`, userID); err != nil {
		return nil, err
	}
	groups := make([]*entity.GoalGroup, 0, len(rows))
	for _, row := range rows {
		group, err := row.ToEntity()
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (r *goalGroupRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) (*entity.GoalGroup, error) {
	var row models.GoalGroupModel
	if err := exec.GetContext(ctx, &row, `SELECT id, user_id, name, sort_order, created_at FROM goal_groups WHERE id = ? AND user_id = ?`, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrGoalGroupNotFound
		}
		return nil, err
	}
	return row.ToEntity()
}

func (r *goalGroupRepository) create(ctx context.Context, exec repository.Executor, group *entity.GoalGroup) error {
	model := models.FromGoalGroupEntity(group)
	result, err := exec.ExecContext(ctx, `INSERT INTO goal_groups (user_id, name, sort_order) VALUES (?, ?, ?)`, model.UserID, model.Name, model.SortOrder)
	if err != nil {
		return wrapGoalGroupDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if id < 0 || id > math.MaxUint32 {
		return fmt.Errorf("goal_groups.id out of range: %d", id)
	}
	group.ID = uint32(id)
	return nil
}

func (r *goalGroupRepository) Save(ctx context.Context, exec repository.Executor, group *entity.GoalGroup) error {
	if group.ID == 0 {
		return r.create(ctx, exec, group)
	}
	model := models.FromGoalGroupEntity(group)
	result, err := exec.ExecContext(ctx, `UPDATE goal_groups SET name = ?, sort_order = ? WHERE id = ? AND user_id = ?`, model.Name, model.SortOrder, model.ID, model.UserID)
	if err != nil {
		return wrapGoalGroupDuplicateError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var exists int
		if err := exec.GetContext(ctx, &exists, `SELECT COUNT(*) FROM goal_groups WHERE id = ? AND user_id = ?`, group.ID, group.UserID); err != nil {
			return err
		}
		if exists == 0 {
			return repository.ErrGoalGroupNotFound
		}
	}
	return nil
}

func (r *goalGroupRepository) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) error {
	result, err := exec.ExecContext(ctx, `DELETE FROM goal_groups WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrGoalGroupNotFound
	}
	return nil
}

func (r *goalGroupRepository) SaveOrder(ctx context.Context, exec repository.Executor, order *entity.GoalGroupOrder) error {
	groups := order.Groups()
	if err := validateGoalGroupOrderPersistence(ctx, exec, order.UserID(), groups); err != nil {
		return err
	}
	if len(groups) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`UPDATE goal_groups SET sort_order = CASE id`)
	args := make([]any, 0, len(groups)*2+1)
	for _, group := range groups {
		query.WriteString(` WHEN ? THEN ?`)
		args = append(args, group.ID, group.SortOrder)
	}
	query.WriteString(` ELSE sort_order END WHERE user_id = ?`)
	args = append(args, order.UserID())
	_, err := exec.ExecContext(ctx, query.String(), args...)
	return err
}

func validateGoalGroupOrderPersistence(ctx context.Context, exec repository.Executor, userID int, groups []*entity.GoalGroup) error {
	var total int
	if err := exec.GetContext(ctx, &total, `SELECT COUNT(*) FROM goal_groups WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if total != len(groups) {
		return repository.ErrGoalGroupOrderInconsistent
	}
	if len(groups) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`SELECT COUNT(*) FROM goal_groups WHERE user_id = ? AND id IN (`)
	args := make([]any, 0, len(groups)+1)
	args = append(args, userID)
	for i, group := range groups {
		if i > 0 {
			query.WriteString(`, `)
		}
		query.WriteString(`?`)
		args = append(args, group.ID)
	}
	query.WriteString(`)`)
	var matched int
	if err := exec.GetContext(ctx, &matched, query.String(), args...); err != nil {
		return err
	}
	if matched != len(groups) {
		return repository.ErrGoalGroupOrderInconsistent
	}
	return nil
}

func (r *goalGroupRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	if err := exec.GetContext(ctx, &count, `SELECT COUNT(*) FROM goal_groups WHERE user_id = ?`, userID); err != nil {
		return 0, err
	}
	return count, nil
}
