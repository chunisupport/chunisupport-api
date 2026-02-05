package chunirec

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/stretchr/testify/assert"
)

func TestCalculateLevel(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{12.0, 12.0},
		{12.4, 12.0},
		{12.5, 12.5},
		{12.9, 12.5},
		{13.0, 13.0},
		{14.8, 14.5},
	}

	for _, test := range tests {
		result := calculateLevel(test.input)
		assert.Equal(t, test.expected, result, "Input: %f", test.input)
	}
}

func TestToMusicShowResponse(t *testing.T) {
	// テスト用のデータを準備
	genreID := 1
	bpm := 180
	releaseDate := time.Date(2023, 4, 13, 0, 0, 0, 0, time.UTC)
	notesVal, _ := notes.NewNotes(500)

	chartConstBAS, _ := chartconstant.NewChartConstant(8.0)
	chartConstMAS, _ := chartconstant.NewChartConstant(13.7)

	song := &repository.SongWithCharts{
		Song: &entity.Song{
			ID:          1,
			DisplayID:   "test-song-001",
			Title:       "テスト楽曲",
			Artist:      "テストアーティスト",
			GenreID:     &genreID,
			BPM:         &bpm,
			ReleasedAt:  &releaseDate,
			OfficialIdx: "001",
			Jacket:      nil,
			IsWorldsend: false,
			IsDeleted:   false,
		},
		Charts: []*entity.Chart{
			{
				ID:             1,
				SongID:         1,
				DifficultyID:   1, // BASIC
				Const:          chartConstBAS,
				IsConstUnknown: false,
				Notes:          &notesVal,
			},
			{
				ID:             2,
				SongID:         1,
				DifficultyID:   4, // MASTER
				Const:          chartConstMAS,
				IsConstUnknown: false,
				Notes:          &notesVal,
			},
		},
	}

	masters := &domainmasterdata.SongMasters{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
		},
	}

	// 変換実行
	result := ToMusicShowResponse(song, masters)

	// 検証
	assert.NotNil(t, result)
	assert.Equal(t, "test-song-001", result.Meta.ID)
	assert.Equal(t, "テスト楽曲", result.Meta.Title)
	assert.Equal(t, "テストアーティスト", result.Meta.Artist)
	assert.NotNil(t, result.Meta.Genre)
	assert.Equal(t, "POPS & ANIME", *result.Meta.Genre)
	assert.NotNil(t, result.Meta.BPM)
	assert.Equal(t, float64(180), *result.Meta.BPM)
	assert.NotNil(t, result.Meta.Release)
	assert.Equal(t, "2023-04-13", *result.Meta.Release)

	// 譜面データの検証
	assert.NotNil(t, result.Data.BAS)
	assert.Equal(t, 8.0, result.Data.BAS.Level)
	assert.Equal(t, 8.0, result.Data.BAS.Const)
	assert.False(t, result.Data.BAS.IsConstUnknown)
	assert.NotNil(t, result.Data.BAS.MaxCombo)
	assert.Equal(t, 500, *result.Data.BAS.MaxCombo)

	assert.NotNil(t, result.Data.MAS)
	assert.Equal(t, 13.5, result.Data.MAS.Level) // 13.7 -> 13.5
	assert.Equal(t, 13.7, result.Data.MAS.Const)
	assert.False(t, result.Data.MAS.IsConstUnknown)

	// 存在しない難易度
	assert.Nil(t, result.Data.ADV)
	assert.Nil(t, result.Data.EXP)
	assert.Nil(t, result.Data.ULT)
}
