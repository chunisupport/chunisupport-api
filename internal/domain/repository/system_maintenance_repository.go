package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SystemMaintenanceRepository はシステム全体のメンテナンス状態を集約単位で永続化します。
type SystemMaintenanceRepository interface {
	Find(ctx context.Context) (*entity.SystemMaintenance, error)
	Save(ctx context.Context, maintenance *entity.SystemMaintenance) error
}
