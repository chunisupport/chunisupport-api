package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type recordFilterRepository struct {
	db *sqlx.DB
}

// NewRecordFilterRepository は新しいRecordFilterRepositoryを生成します。
func NewRecordFilterRepository(db *sqlx.DB) repository.RecordFilterRepository {
	return &recordFilterRepository{db: db}
}

func (r *recordFilterRepository) ListByUserID(ctx context.Context, userID int) ([]*entity.RecordFilter, error) {
	var filterModels []*models.RecordFilterModel
	query := `SELECT id, user_id, name, filter_value_gzip, is_worldsend, updated_at FROM record_filters WHERE user_id = ? ORDER BY updated_at DESC, id ASC`
	if err := r.db.SelectContext(ctx, &filterModels, query, userID); err != nil {
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

func (r *recordFilterRepository) FindByIDAndUserID(ctx context.Context, id []byte, userID int) (*entity.RecordFilter, error) {
	var filterModel models.RecordFilterModel
	query := `SELECT id, user_id, name, filter_value_gzip, is_worldsend, updated_at FROM record_filters WHERE id = ? AND user_id = ?`
	if err := r.db.GetContext(ctx, &filterModel, query, id, userID); err != nil {
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

func (r *recordFilterRepository) Save(ctx context.Context, filter *entity.RecordFilter) error {
	query := `
INSERT INTO record_filters (id, user_id, name, filter_value_gzip, is_worldsend, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    filter_value_gzip = excluded.filter_value_gzip,
    is_worldsend = excluded.is_worldsend,
    updated_at = CURRENT_TIMESTAMP
WHERE record_filters.user_id = excluded.user_id
`
	result, err := r.db.ExecContext(ctx, query, filter.ID(), filter.UserID(), filter.Name(), filter.FilterValueGzip(), filter.IsWorldsend())
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

func (r *recordFilterRepository) DeleteByIDAndUserID(ctx context.Context, id []byte, userID int) error {
	query := `DELETE FROM record_filters WHERE id = ? AND user_id = ?`
	result, err := r.db.ExecContext(ctx, query, id, userID)
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

func (r *recordFilterRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM record_filters WHERE user_id = ?`, userID); err != nil {
		return 0, err
	}
	return count, nil
}

var _ repository.RecordFilterRepository = (*recordFilterRepository)(nil)
