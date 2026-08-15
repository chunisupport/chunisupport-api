package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerMetricHistoryModel は公式RATING・公式OVER POWER履歴の永続化モデルです。
type PlayerMetricHistoryModel struct {
	PlayerID          int       `db:"player_id"`
	OfficialRating    float64   `db:"official_rating"`
	OfficialOverpower float64   `db:"official_overpower"`
	DataCollectedAt   time.Time `db:"data_collected_at"`
}

// PlayerMetricHistoryModelFromEntity は履歴エンティティを永続化モデルへ変換します。
func PlayerMetricHistoryModelFromEntity(entry entity.PlayerMetricHistoryEntry) PlayerMetricHistoryModel {
	return PlayerMetricHistoryModel{
		PlayerID: entry.PlayerID, OfficialRating: entry.OfficialRating,
		OfficialOverpower: entry.OfficialOverpower, DataCollectedAt: entry.DataCollectedAt,
	}
}

// ToEntity は永続化モデルを履歴エンティティへ変換します。
func (m PlayerMetricHistoryModel) ToEntity() entity.PlayerMetricHistoryEntry {
	return entity.PlayerMetricHistoryEntry{
		PlayerID: m.PlayerID, OfficialRating: m.OfficialRating,
		OfficialOverpower: m.OfficialOverpower, DataCollectedAt: m.DataCollectedAt,
	}
}
