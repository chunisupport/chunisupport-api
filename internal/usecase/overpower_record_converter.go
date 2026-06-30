package usecase

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
)

// playerRecordsToOverpowerRecordsはPlayerRecordの集合をOVER POWER集計用レコードへ変換する。
// failOnMissingRelatedがtrueの場合、関連エンティティ欠損を不整合としてエラーを返し、
// falseの場合は欠損レコードを安全にスキップする。
func playerRecordsToOverpowerRecords(records []*entity.PlayerRecord, failOnMissingRelated bool, include func(*entity.PlayerRecord) bool) ([]domainservice.OverpowerRecord, error) {
	overpowerRecords := make([]domainservice.OverpowerRecord, 0, len(records))
	for i, record := range records {
		if record == nil {
			if failOnMissingRelated {
				return nil, fmt.Errorf("player record is nil at index=%d", i)
			}
			continue
		}
		if record.Song == nil {
			if failOnMissingRelated {
				return nil, fmt.Errorf("song is nil in player record at index=%d", i)
			}
			continue
		}
		if record.Chart == nil {
			if failOnMissingRelated {
				return nil, fmt.Errorf("chart is nil in player record at index=%d", i)
			}
			continue
		}
		if include != nil && !include(record) {
			continue
		}

		overpowerRecords = append(overpowerRecords, domainservice.OverpowerRecord{
			SongID:      record.Song.ID,
			Score:       uint32(record.Score),
			ChartConst:  record.Chart.Const.Float64(),
			ComboLampID: record.ComboLampID,
		})
	}
	return overpowerRecords, nil
}
