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
	overpowerRecords, _, err := playerRecordsToOverpowerRecordsWithSkipped(records, failOnMissingRelated, func(record *entity.PlayerRecord) (bool, string) {
		if include == nil || include(record) {
			return true, ""
		}
		return false, "excluded"
	})
	return overpowerRecords, err
}

type skippedOverpowerRecord struct {
	Index     int    `json:"index"`
	PlayerID  int    `json:"player_id,omitempty"`
	SongID    int    `json:"song_id,omitempty"`
	SongTitle string `json:"song_title,omitempty"`
	ChartID   int    `json:"chart_id,omitempty"`
	Reason    string `json:"reason"`
}

func playerRecordsToOverpowerRecordsWithSkipped(records []*entity.PlayerRecord, failOnMissingRelated bool, include func(*entity.PlayerRecord) (bool, string)) ([]domainservice.OverpowerRecord, []skippedOverpowerRecord, error) {
	overpowerRecords := make([]domainservice.OverpowerRecord, 0, len(records))
	skipped := make([]skippedOverpowerRecord, 0)
	for i, record := range records {
		if record == nil {
			if failOnMissingRelated {
				return nil, nil, fmt.Errorf("player record is nil at index=%d", i)
			}
			skipped = append(skipped, skippedOverpowerRecord{Index: i, Reason: "record_nil"})
			continue
		}
		if record.Song == nil {
			if failOnMissingRelated {
				return nil, nil, fmt.Errorf("song is nil in player record at index=%d", i)
			}
			skipped = append(skipped, newSkippedOverpowerRecord(i, record, "song_nil"))
			continue
		}
		if record.Chart == nil {
			if failOnMissingRelated {
				return nil, nil, fmt.Errorf("chart is nil in player record at index=%d", i)
			}
			skipped = append(skipped, newSkippedOverpowerRecord(i, record, "chart_nil"))
			continue
		}
		if include != nil {
			included, reason := include(record)
			if !included {
				if reason == "" {
					reason = "excluded"
				}
				skipped = append(skipped, newSkippedOverpowerRecord(i, record, reason))
				continue
			}
		}

		overpowerRecords = append(overpowerRecords, domainservice.OverpowerRecord{
			SongID:      record.Song.ID,
			Score:       uint32(record.Score),
			ChartConst:  record.Chart.Const.Float64(),
			ComboLampID: record.ComboLampID,
		})
	}
	return overpowerRecords, skipped, nil
}

func newSkippedOverpowerRecord(index int, record *entity.PlayerRecord, reason string) skippedOverpowerRecord {
	skipped := skippedOverpowerRecord{Index: index, Reason: reason}
	if record == nil {
		return skipped
	}
	skipped.PlayerID = record.PlayerID
	skipped.ChartID = record.ChartID
	if record.Song != nil {
		skipped.SongID = record.Song.ID
		skipped.SongTitle = record.Song.Title
	}
	return skipped
}
