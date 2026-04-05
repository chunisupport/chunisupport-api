package service

import (
	"log/slog"
	"math"
	"sort"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"

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
// difficultySortOrderByID はID→SortOrderのマップで、ゲームの正規表示順（BASIC<ADVANCED<EXPERT<MASTER<ULTIMA）でソートするために使用します。
func (s *RecordCompletionService) CompletePlayerRecords(records []*entity.PlayerRecord, songs []*entity.Song, difficultyNamesByID map[int]string, difficultySortOrderByID map[int]int) []*entity.PlayerRecord {
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
				ChartDifficulty: &master.ChartDifficulty{
					ID:        chart.DifficultyID,
					Name:      difficultyNameByID(chart.DifficultyID, difficultyNamesByID),
					SortOrder: difficultySortOrder(chart.DifficultyID, difficultySortOrderByID),
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
		return playerRecordDifficultySortOrder(completed[i], difficultySortOrderByID) < playerRecordDifficultySortOrder(completed[j], difficultySortOrderByID)
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

// difficultySortOrder はID→SortOrderのマップから難易度ソート順を返します。
// マスタデータに該当IDが存在しない場合は math.MaxInt を返し、データ不整合を検知するため警告ログを出力します。
func difficultySortOrder(difficultyID int, difficultySortOrderByID map[int]int) int {
	if order, ok := difficultySortOrderByID[difficultyID]; ok {
		return order
	}
	slog.Warn("マスタデータに難易度ソート順が見つかりません", "difficulty_id", difficultyID)
	return math.MaxInt
}

func playerRecordSongID(record *entity.PlayerRecord) int {
	if record == nil || record.Song == nil {
		return math.MaxInt
	}
	return record.Song.ID
}

// playerRecordDifficultySortOrder はレコードの難易度SortOrderを返します。
// 未知の難易度IDの場合はmath.MaxIntを返し、末尾に回します。
func playerRecordDifficultySortOrder(record *entity.PlayerRecord, difficultySortOrderByID map[int]int) int {
	if record == nil || record.Chart == nil {
		return math.MaxInt
	}
	order, ok := difficultySortOrderByID[record.Chart.DifficultyID]
	if !ok {
		return math.MaxInt
	}
	return order
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
