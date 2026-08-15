package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

const systemMaintenanceColumns = "id, enabled, comment, updated_by_user_id, updated_at"

type systemMaintenanceRepository struct {
	db *sqlx.DB
}

// NewSystemMaintenanceRepository はメンテナンス状態リポジトリを生成します。
func NewSystemMaintenanceRepository(db *sqlx.DB) domainrepo.SystemMaintenanceRepository {
	return &systemMaintenanceRepository{db: db}
}

// Find はsingleton行を取得し、ドメインの不変条件を検証して復元します。
func (r *systemMaintenanceRepository) Find(ctx context.Context) (*entity.SystemMaintenance, error) {
	var model models.SystemMaintenanceModel
	query := `SELECT ` + systemMaintenanceColumns + ` FROM system_maintenance WHERE id = ?`
	if err := r.db.GetContext(ctx, &model, query, entity.SystemMaintenanceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrSystemMaintenanceNotFound
		}
		return nil, err
	}
	return model.ToEntity()
}

// Save はメンテナンス状態の集約全体を既存のsingleton行へ保存します。
func (r *systemMaintenanceRepository) Save(ctx context.Context, maintenance *entity.SystemMaintenance) error {
	model := models.FromSystemMaintenanceEntity(maintenance)
	result, err := r.db.ExecContext(ctx, `
UPDATE system_maintenance
SET enabled = ?, comment = ?, updated_by_user_id = ?, updated_at = ?
WHERE id = ?
`, model.Enabled, model.Comment, model.UpdatedByUserID, model.UpdatedAt, model.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}

	var exists int
	if err := r.db.GetContext(
		ctx,
		&exists,
		`SELECT COUNT(*) FROM system_maintenance WHERE id = ?`,
		model.ID,
	); err != nil {
		return err
	}
	if exists == 0 {
		return domainrepo.ErrSystemMaintenanceNotFound
	}
	return nil
}

var _ domainrepo.SystemMaintenanceRepository = (*systemMaintenanceRepository)(nil)
