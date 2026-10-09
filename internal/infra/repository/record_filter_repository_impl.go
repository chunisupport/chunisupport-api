package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
)

type recordFilterRepository struct{}

// NewRecordFilterRepository は新しいRecordFilterRepositoryを生成します。
func NewRecordFilterRepository() repository.RecordFilterRepository {
	return &recordFilterRepository{}
}

func (r *recordFilterRepository) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.RecordFilter, error) {
	var filterModels []*models.RecordFilterModel
	query := `SELECT id, user_id, name, filter_value_gzip, is_worldsend, created_at, updated_at FROM record_filters WHERE user_id = ? ORDER BY updated_at DESC, id ASC`
	if err := exec.SelectContext(ctx, &filterModels, query, userID); err != nil {
		return nil, err
	}
	filters := make([]*entity.RecordFilter, 0, len(filterModels))
	for _, m := range filterModels {
		filter, err := m.ToEntity()
		if err != nil {
			return nil, errors.Join(repository.ErrRepositoryOperationFailed, err)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func (r *recordFilterRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id []byte, userID int) (*entity.RecordFilter, error) {
	var filterModel models.RecordFilterModel
	query := `SELECT id, user_id, name, filter_value_gzip, is_worldsend, created_at, updated_at FROM record_filters WHERE id = ? AND user_id = ?`
	if err := exec.GetContext(ctx, &filterModel, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrRecordFilterNotFound, err)
		}
		return nil, err
	}
	filter, err := filterModel.ToEntity()
	if err != nil {
		return nil, errors.Join(repository.ErrRepositoryOperationFailed, err)
	}
	return filter, nil
}

func (r *recordFilterRepository) Create(ctx context.Context, exec repository.Executor, filter *entity.RecordFilter) error {
	query := `
INSERT INTO record_filters (id, user_id, name, filter_value_gzip, is_worldsend, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`
	_, err := exec.ExecContext(ctx, query, filter.ID(), filter.UserID(), filter.Name(), filter.FilterValueGzip(), filter.IsWorldsend())
	return err
}

// Update は所有者を限定して既存のフィルタを更新します。
// 本番のDSNは clientFoundRows=true のため、値が変わらない UPDATE でもマッチした行数が返り、存在判定に使えます。
func (r *recordFilterRepository) Update(ctx context.Context, exec repository.Executor, filter *entity.RecordFilter) error {
	query := `
UPDATE record_filters
SET name = ?, filter_value_gzip = ?, is_worldsend = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
`
	result, err := exec.ExecContext(ctx, query, filter.Name(), filter.FilterValueGzip(), filter.IsWorldsend(), filter.ID(), filter.UserID())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrRecordFilterNotFound
	}
	return nil
}

func (r *recordFilterRepository) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id []byte, userID int) error {
	query := `DELETE FROM record_filters WHERE id = ? AND user_id = ?`
	result, err := exec.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrRecordFilterNotFound
	}
	return nil
}

func (r *recordFilterRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	if err := exec.GetContext(ctx, &count, `SELECT COUNT(*) FROM record_filters WHERE user_id = ?`, userID); err != nil {
		return 0, err
	}
	return count, nil
}

var _ repository.RecordFilterRepository = (*recordFilterRepository)(nil)
