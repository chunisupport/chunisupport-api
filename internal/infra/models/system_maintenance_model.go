package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SystemMaintenanceModel はメンテナンス状態のデータベースモデルです。
type SystemMaintenanceModel struct {
	ID              int       `db:"id"`
	Enabled         bool      `db:"enabled"`
	Comment         string    `db:"comment"`
	UpdatedByUserID *int      `db:"updated_by_user_id"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// ToEntity は永続化モデルを不変条件が検証されたメンテナンス状態へ変換します。
func (m *SystemMaintenanceModel) ToEntity() (*entity.SystemMaintenance, error) {
	return entity.RestoreSystemMaintenance(
		m.ID,
		m.Enabled,
		m.Comment,
		m.UpdatedByUserID,
		m.UpdatedAt,
	)
}

// FromSystemMaintenanceEntity はドメインエンティティをデータベースモデルへ変換します。
func FromSystemMaintenanceEntity(maintenance *entity.SystemMaintenance) *SystemMaintenanceModel {
	return &SystemMaintenanceModel{
		ID:              maintenance.ID,
		Enabled:         maintenance.Enabled,
		Comment:         maintenance.Comment.String(),
		UpdatedByUserID: maintenance.UpdatedByUserID,
		UpdatedAt:       maintenance.UpdatedAt,
	}
}
