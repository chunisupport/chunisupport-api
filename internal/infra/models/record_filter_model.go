package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecordFilterModel はデータベース用のRecordFilterモデルです。
type RecordFilterModel struct {
	ID              []byte    `db:"id"`
	UserID          int       `db:"user_id"`
	Name            string    `db:"name"`
	FilterValueGzip []byte    `db:"filter_value_gzip"`
	IsWorldsend     bool      `db:"is_worldsend"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func (m *RecordFilterModel) ToEntity() (*entity.RecordFilter, error) {
	return entity.RestoreRecordFilter(m.ID, m.UserID, m.Name, m.FilterValueGzip, m.IsWorldsend, m.CreatedAt, m.UpdatedAt)
}

func FromRecordFilterEntity(e *entity.RecordFilter) *RecordFilterModel {
	return &RecordFilterModel{
		ID:              e.ID(),
		UserID:          e.UserID(),
		Name:            e.Name(),
		FilterValueGzip: e.FilterValueGzip(),
		IsWorldsend:     e.IsWorldsend(),
		CreatedAt:       e.CreatedAt(),
		UpdatedAt:       e.UpdatedAt(),
	}
}
