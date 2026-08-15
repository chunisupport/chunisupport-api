package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerLatestUpdateModel は最新データ登録結果のデータベースモデルです。
type PlayerLatestUpdateModel struct {
	PlayerID        int       `db:"player_id"`
	SchemaVersion   int       `db:"schema_version"`
	ResultGzip      []byte    `db:"result_gzip"`
	SourceUpdatedAt time.Time `db:"source_updated_at"`
	ImportedAt      time.Time `db:"imported_at"`
	BodyHash        string    `db:"body_hash"`
}

// FromPlayerLatestUpdateEntity はドメインエンティティをデータベースモデルへ変換します。
func FromPlayerLatestUpdateEntity(update *entity.PlayerLatestUpdate) *PlayerLatestUpdateModel {
	return &PlayerLatestUpdateModel{
		PlayerID:        update.PlayerID(),
		SchemaVersion:   update.SchemaVersion(),
		ResultGzip:      update.ResultGzip(),
		SourceUpdatedAt: update.SourceUpdatedAt(),
		ImportedAt:      update.ImportedAt(),
		BodyHash:        update.BodyHash(),
	}
}

// ToEntity はデータベースモデルをドメインエンティティへ変換します。
func (m *PlayerLatestUpdateModel) ToEntity() (*entity.PlayerLatestUpdate, error) {
	return entity.NewPlayerLatestUpdate(
		m.PlayerID,
		m.SchemaVersion,
		m.ResultGzip,
		m.SourceUpdatedAt,
		m.ImportedAt,
		m.BodyHash,
	)
}
