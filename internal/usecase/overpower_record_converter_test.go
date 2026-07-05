package usecase

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerRecordsToOverpowerRecordsWithSkipped(t *testing.T) {
	// Given
	recordScore, err := score.NewScore(1_000_000)
	require.NoError(t, err)
	chartConst, err := chartconstant.NewChartConstant(10.0)
	require.NoError(t, err)

	records := []*entity.PlayerRecord{
		nil,
		{PlayerID: 1, ChartID: 10, Score: recordScore, Chart: &entity.Chart{ID: 10, Const: chartConst}},
		{PlayerID: 1, ChartID: 11, Score: recordScore, Song: &entity.Song{ID: 100, Title: "テスト曲"}},
		{PlayerID: 1, ChartID: 12, Score: recordScore, Song: &entity.Song{ID: 101, Title: "除外曲"}, Chart: &entity.Chart{ID: 12, Const: chartConst}},
		{PlayerID: 1, ChartID: 13, Score: recordScore, Song: &entity.Song{ID: 102, Title: "集計曲"}, Chart: &entity.Chart{ID: 13, Const: chartConst}},
	}

	// When
	overpowerRecords, skipped, err := playerRecordsToOverpowerRecordsWithSkipped(records, false, func(record *entity.PlayerRecord) (bool, string) {
		if record.Song.ID == 101 {
			return false, "locked_song"
		}
		return true, ""
	})

	// Then
	require.NoError(t, err)
	assert.Len(t, overpowerRecords, 1)
	assert.Equal(t, []skippedOverpowerRecord{
		{Index: 0, Reason: "record_nil"},
		{Index: 1, PlayerID: 1, ChartID: 10, Reason: "song_nil"},
		{Index: 2, PlayerID: 1, SongID: 100, SongTitle: "テスト曲", ChartID: 11, Reason: "chart_nil"},
		{Index: 3, PlayerID: 1, SongID: 101, SongTitle: "除外曲", ChartID: 12, Reason: "locked_song"},
	}, skipped)
}
