package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

func TestRecordCompletionService_CompletePlayerRecords(t *testing.T) {
	svc := NewRecordCompletionService()

	songs := []*entity.Song{
		{
			ID: 1,
			Charts: []*entity.Chart{
				{ID: 11, SongID: 1, DifficultyID: 3},
				{ID: 12, SongID: 1, DifficultyID: 4},
			},
		},
		{
			ID: 2,
			Charts: []*entity.Chart{
				{ID: 21, SongID: 2, DifficultyID: 2},
			},
		},
		{
			ID:        3,
			IsDeleted: true,
			Charts: []*entity.Chart{
				{ID: 31, SongID: 3, DifficultyID: 4},
			},
		},
	}

	records := []*entity.PlayerRecord{
		{
			ChartID: 11,
			Song:    songs[0],
			Chart:   songs[0].Charts[0],
			ChartDifficulty: &master.ChartDifficulty{
				ID:   3,
				Name: "EXPERT",
			},
		},
	}

	completed := svc.CompletePlayerRecords(records, songs, map[int]string{2: "ADVANCED", 4: "MASTER"}, map[int]int{2: 1, 3: 2, 4: 3})

	if len(completed) != 3 {
		require.Failf(t, "前提条件失敗", "expected 3 records, got %d", len(completed))
	}
	if completed[0].ChartID != 11 || completed[1].ChartID != 12 || completed[2].ChartID != 21 {
		require.Failf(t, "前提条件失敗", "unexpected sorted chart ids: %d, %d, %d", completed[0].ChartID, completed[1].ChartID, completed[2].ChartID)
	}
	if completed[1].ChartDifficulty == nil || completed[1].ChartDifficulty.Name != "MASTER" {
		require.Failf(t, "前提条件失敗", "expected completed difficulty MASTER, got %+v", completed[1].ChartDifficulty)
	}
	if completed[2].ChartDifficulty == nil || completed[2].ChartDifficulty.Name != "ADVANCED" {
		require.Failf(t, "前提条件失敗", "expected completed difficulty ADVANCED, got %+v", completed[2].ChartDifficulty)
	}
}

func TestRecordCompletionService_CompleteWorldsendRecords(t *testing.T) {
	svc := NewRecordCompletionService()

	song1 := &entity.Song{ID: 1}
	song2 := &entity.Song{ID: 2}
	song3 := &entity.Song{ID: 3, IsDeleted: true}
	chart1 := &entity.WorldsendChart{ID: 101, SongID: 1}
	chart2 := &entity.WorldsendChart{ID: 102, SongID: 2}
	chart3 := &entity.WorldsendChart{ID: 103, SongID: 3}

	records := []*entity.PlayerWorldsendRecord{
		{WorldsendChartID: 101, Song: song1, WorldsendChart: chart1},
	}
	songCharts := []*entity.WorldsendSongWithChart{
		{Song: song1, Chart: chart1},
		{Song: song2, Chart: chart2},
		{Song: song3, Chart: chart3},
	}

	completed := svc.CompleteWorldsendRecords(records, songCharts)

	if len(completed) != 2 {
		require.Failf(t, "前提条件失敗", "expected 2 records, got %d", len(completed))
	}
	if completed[0].WorldsendChartID != 101 || completed[1].WorldsendChartID != 102 {
		require.Failf(t, "前提条件失敗", "unexpected sorted worldsend chart ids: %d, %d", completed[0].WorldsendChartID, completed[1].WorldsendChartID)
	}
}

// 未知の難易度IDの譜面が既知の譜面より後ろにソートされることを検証する。
// 未知譜面のChartID(1)は既知譜面のChartID(4)より小さいため、ID順では説明できない順序で検証する。
func TestRecordCompletionService_CompletePlayerRecords_UnknownDifficultySortsLast(t *testing.T) {
	// Given
	svc := NewRecordCompletionService()
	song := &entity.Song{ID: 1}
	knownChart := &entity.Chart{ID: 4, SongID: 1, DifficultyID: 4}
	unknownChart := &entity.Chart{ID: 1, SongID: 1, DifficultyID: 1}
	records := []*entity.PlayerRecord{
		{ChartID: 1, Song: song, Chart: unknownChart, ChartDifficulty: &master.ChartDifficulty{ID: 1}},
		{ChartID: 4, Song: song, Chart: knownChart, ChartDifficulty: &master.ChartDifficulty{ID: 4, Name: "MASTER"}},
	}

	// When
	completed := svc.CompletePlayerRecords(records, nil, map[int]string{4: "MASTER"}, map[int]int{4: 3})

	// Then
	require.Len(t, completed, 2)
	assert.Equal(t, []int{4, 1}, []int{completed[0].ChartID, completed[1].ChartID})
}

// 同一楽曲のWORLD'S END譜面が譜面ID昇順にソートされることを検証する。
// 既プレイ(202)を先に入力し未プレイ(201)を補完させることで、入力逆順がソートで修正されることを検証する。
func TestRecordCompletionService_CompleteWorldsendRecords_SameSongSortsByChartID(t *testing.T) {
	// Given
	svc := NewRecordCompletionService()
	song := &entity.Song{ID: 1}
	chartLow := &entity.WorldsendChart{ID: 201, SongID: 1}
	chartHigh := &entity.WorldsendChart{ID: 202, SongID: 1}
	records := []*entity.PlayerWorldsendRecord{
		{WorldsendChartID: 202, Song: song, WorldsendChart: chartHigh},
	}
	songCharts := []*entity.WorldsendSongWithChart{
		{Song: song, Chart: chartHigh},
		{Song: song, Chart: chartLow},
	}

	// When
	completed := svc.CompleteWorldsendRecords(records, songCharts)

	// Then
	require.Len(t, completed, 2)
	assert.Equal(t, []int{201, 202}, []int{completed[0].WorldsendChartID, completed[1].WorldsendChartID})
}
