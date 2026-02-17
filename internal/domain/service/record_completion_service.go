package service

import (
	"math"
	"sort"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// WorldsendSongChartPair は WORLD'S END 楽曲と譜面の組を表します。
type WorldsendSongChartPair struct {
	Song  *entity.Song
	Chart *entity.WorldsendChart
}

// RecordCompletionService は未プレイレコード補完を行うドメインサービスです。
type RecordCompletionService struct{}

// NewRecordCompletionService は RecordCompletionService を生成します。
func NewRecordCompletionService() *RecordCompletionService {
	return &RecordCompletionService{}
}

// CompletePlayerRecords は通常譜面レコードを未プレイ補完し、ソート済み配列を返します。
func (s *RecordCompletionService) CompletePlayerRecords(records []*entity.PlayerRecord, songs []*entity.Song, difficultyNamesByID map[int]string) []*entity.PlayerRecord {
	completed := make([]*entity.PlayerRecord, 0, len(records))
	playedByChartID := make(map[int]struct{}, len(records))

	for _, record := range records {
		if record == nil {
			continue
		}
		playedByChartID[record.ChartID] = struct{}{}
		completed = append(completed, record)
	}

	for _, song := range songs {
		if song == nil || song.IsDeleted {
			continue
		}
		for _, chart := range song.Charts {
			if chart == nil {
				continue
			}
			if _, ok := playedByChartID[chart.ID]; ok {
				continue
			}
			completed = append(completed, &entity.PlayerRecord{
				ChartID: chart.ID,
				Chart:   chart,
				Song:    song,
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   chart.DifficultyID,
					Name: difficultyNameByID(chart.DifficultyID, difficultyNamesByID),
				},
			})
		}
	}

	sort.Slice(completed, func(i, j int) bool {
		leftSongID := playerRecordSongID(completed[i])
		rightSongID := playerRecordSongID(completed[j])
		if leftSongID != rightSongID {
			return leftSongID < rightSongID
		}
		return playerRecordDifficultyID(completed[i]) < playerRecordDifficultyID(completed[j])
	})

	return completed
}

// CompleteWorldsendRecords は WORLD'S END レコードを未プレイ補完し、ソート済み配列を返します。
func (s *RecordCompletionService) CompleteWorldsendRecords(records []*entity.PlayerWorldsendRecord, songCharts []*WorldsendSongChartPair) []*entity.PlayerWorldsendRecord {
	completed := make([]*entity.PlayerWorldsendRecord, 0, len(records))
	playedByChartID := make(map[int]struct{}, len(records))

	for _, record := range records {
		if record == nil {
			continue
		}
		playedByChartID[record.WorldsendChartID] = struct{}{}
		completed = append(completed, record)
	}

	for _, songChart := range songCharts {
		if songChart == nil || songChart.Song == nil || songChart.Chart == nil || songChart.Song.IsDeleted {
			continue
		}
		if _, ok := playedByChartID[songChart.Chart.ID]; ok {
			continue
		}
		completed = append(completed, &entity.PlayerWorldsendRecord{
			WorldsendChartID: songChart.Chart.ID,
			Song:             songChart.Song,
			WorldsendChart:   songChart.Chart,
		})
	}

	sort.Slice(completed, func(i, j int) bool {
		leftSongID := worldsendRecordSongID(completed[i])
		rightSongID := worldsendRecordSongID(completed[j])
		if leftSongID != rightSongID {
			return leftSongID < rightSongID
		}
		return worldsendRecordChartID(completed[i]) < worldsendRecordChartID(completed[j])
	})

	return completed
}

func difficultyNameByID(difficultyID int, difficultyNamesByID map[int]string) string {
	if difficultyNamesByID == nil {
		return ""
	}
	return difficultyNamesByID[difficultyID]
}

func playerRecordSongID(record *entity.PlayerRecord) int {
	if record == nil || record.Song == nil {
		return math.MaxInt
	}
	return record.Song.ID
}

func playerRecordDifficultyID(record *entity.PlayerRecord) int {
	if record == nil || record.Chart == nil {
		return math.MaxInt
	}
	return record.Chart.DifficultyID
}

func worldsendRecordSongID(record *entity.PlayerWorldsendRecord) int {
	if record == nil || record.Song == nil {
		return math.MaxInt
	}
	return record.Song.ID
}

func worldsendRecordChartID(record *entity.PlayerWorldsendRecord) int {
	if record == nil {
		return math.MaxInt
	}
	if record.WorldsendChart != nil {
		return record.WorldsendChart.ID
	}
	return record.WorldsendChartID
}
