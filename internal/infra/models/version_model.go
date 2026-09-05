package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// VersionModel はversionsテーブルの永続化モデルです。
type VersionModel struct {
	ID         int       `db:"id"`
	Name       string    `db:"name"`
	ReleasedAt time.Time `db:"released_at"`
}

// ToEntity は永続化モデルをエンティティへ変換します。
func (m *VersionModel) ToEntity() *entity.Version {
	releasedAt := time.Date(m.ReleasedAt.Year(), m.ReleasedAt.Month(), m.ReleasedAt.Day(), 0, 0, 0, 0, time.UTC)
	return &entity.Version{ID: m.ID, Name: m.Name, ReleasedAt: releasedAt}
}

// FromEntity はエンティティを永続化モデルへ変換します。
func FromVersionEntity(version *entity.Version) *VersionModel {
	return &VersionModel{ID: version.ID, Name: version.Name, ReleasedAt: version.ReleasedAt}
}
