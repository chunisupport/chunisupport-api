package service

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculatePlayerRecordStatistics_全体と難易度別に累積集計する(t *testing.T) {
	// Given
	records := []*entity.PlayerRecord{
		newStatisticsRecord(t, "BASIC", 1010000, "CLEAR", "ALL JUSTICE", "FULL CHAIN GOLD"),
		newStatisticsRecord(t, "MASTER", 1008999, "FAILED", "FULL COMBO", "NONE"),
		newStatisticsRecord(t, "ULTIMA", 1000000, "CLEAR", "NONE", "FULL CHAIN PLATINUM"),
	}

	// When
	result, err := CalculatePlayerRecordStatistics(records)

	// Then
	require.NoError(t, err)
	assert.Equal(t, int64(3018999), result.Overall.TotalHighScore)
	assert.Equal(t, RecordAchievementStatistics{AJ: 1, FC: 2, CLR: 2, FCH: 2, MAX: 1, SSSPlus: 1, SSS: 2, SSPlus: 2, SS: 3, SPlus: 3, S: 3}, result.Overall.Achievements)
	assert.Equal(t, int64(1010000), result.ByDifficulty["BASIC"].TotalHighScore)
	assert.Equal(t, int64(1008999), result.ByDifficulty["MASTER"].TotalHighScore)
	assert.Len(t, result.ByDifficulty, 5)
	assert.Zero(t, result.ByDifficulty["ADVANCED"].TotalHighScore)
}

func TestCalculatePlayerRecordStatistics_スコア境界値を集計する(t *testing.T) {
	tests := []struct {
		name     string
		score    uint32
		expected RecordAchievementStatistics
	}{
		{name: "MAX", score: 1010000, expected: RecordAchievementStatistics{MAX: 1, SSSPlus: 1, SSS: 1, SSPlus: 1, SS: 1, SPlus: 1, S: 1}},
		{name: "SSS+境界", score: 1009000, expected: RecordAchievementStatistics{SSSPlus: 1, SSS: 1, SSPlus: 1, SS: 1, SPlus: 1, S: 1}},
		{name: "SSS境界", score: 1007500, expected: RecordAchievementStatistics{SSS: 1, SSPlus: 1, SS: 1, SPlus: 1, S: 1}},
		{name: "SS+境界", score: 1005000, expected: RecordAchievementStatistics{SSPlus: 1, SS: 1, SPlus: 1, S: 1}},
		{name: "SS境界", score: 1000000, expected: RecordAchievementStatistics{SS: 1, SPlus: 1, S: 1}},
		{name: "S+境界", score: 990000, expected: RecordAchievementStatistics{SPlus: 1, S: 1}},
		{name: "S境界", score: 975000, expected: RecordAchievementStatistics{S: 1}},
		{name: "S未満", score: 974999, expected: RecordAchievementStatistics{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			record := newStatisticsRecord(t, "EXPERT", tt.score, "FAILED", "NONE", "NONE")

			// When
			result, err := CalculatePlayerRecordStatistics([]*entity.PlayerRecord{record})

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Overall.Achievements)
		})
	}
}

func TestCalculatePlayerRecordStatistics_不正な関連情報を拒否する(t *testing.T) {
	tests := []struct {
		name   string
		record *entity.PlayerRecord
	}{
		{name: "nilレコード", record: nil},
		{name: "未知の難易度", record: newStatisticsRecord(t, "WORLD'S END", 1000000, "CLEAR", "NONE", "NONE")},
		{name: "コンボランプ欠損", record: func() *entity.PlayerRecord {
			r := newStatisticsRecord(t, "MASTER", 1000000, "CLEAR", "NONE", "NONE")
			r.ComboLamp = nil
			return r
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := CalculatePlayerRecordStatistics([]*entity.PlayerRecord{tt.record})

			// Then
			assert.Error(t, err)
		})
	}
}

func newStatisticsRecord(t *testing.T, difficulty string, value uint32, clearLamp string, comboLamp string, fullChain string) *entity.PlayerRecord {
	t.Helper()
	recordScore, err := score.NewScore(value)
	require.NoError(t, err)
	return &entity.PlayerRecord{
		Score:           recordScore,
		ChartDifficulty: &entity.ChartDifficulty{Name: difficulty},
		ClearLamp:       &entity.ClearLampType{Name: clearLamp},
		ComboLamp:       &entity.ComboLampType{Name: comboLamp},
		FullChain:       &entity.FullChainType{Name: fullChain},
	}
}
