package usecase

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreRanker_Next(t *testing.T) {
	tests := []struct {
		name string
		// Given: スコア降順に並んだスコア
		scores []uint32
		// Then: 期待する順位
		expected []int
	}{
		{
			name:     "同点がない場合は連番になる",
			scores:   []uint32{1_010_000, 1_009_000, 1_008_000},
			expected: []int{1, 2, 3},
		},
		{
			name:     "途中の同点も同順位にし次の順位を人数分飛ばす",
			scores:   []uint32{1_010_000, 1_009_000, 1_009_000, 1_008_000},
			expected: []int{1, 2, 2, 4},
		},
		{
			name:     "先頭のスコアが0でも1位から数える",
			scores:   []uint32{0, 0},
			expected: []int{1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var ranker scoreRanker

			// When
			got := make([]int, 0, len(tt.scores))
			for _, score := range tt.scores {
				got = append(got, ranker.Next(score))
			}

			// Then
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNewChartRankingEntry(t *testing.T) {
	chartConst, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	updatedAt := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	clearLamp := "CLEAR"
	allJustice := "ALL JUSTICE"
	fullCombo := "FULL COMBO"

	tests := []struct {
		name string
		// Given
		chart     ChartRankingChart
		comboLamp string
		// Then
		expected ChartRankingEntry
	}{
		{
			name:      "ALL JUSTICEはコンボランプ補正込みでレーティングとOPを計算する",
			chart:     ChartRankingChart{Const: &chartConst},
			comboLamp: "ALL JUSTICE",
			expected: ChartRankingEntry{
				Rank: 2, Username: "user", PlayerName: "PLAYER", Score: 1_008_500,
				Rating: 16.6, Overpower: 85.0, OverpowerPercent: 97.14285,
				ClearLamp: &clearLamp, ComboLamp: &allJustice, UpdatedAt: updatedAt,
			},
		},
		{
			name:      "FULL COMBOはコンボランプ補正込みでレーティングとOPを計算する",
			chart:     ChartRankingChart{Const: &chartConst},
			comboLamp: "FULL COMBO",
			expected: ChartRankingEntry{
				Rank: 2, Username: "user", PlayerName: "PLAYER", Score: 1_008_500,
				Rating: 16.6, Overpower: 84.5, OverpowerPercent: 96.57142,
				ClearLamp: &clearLamp, ComboLamp: &fullCombo, UpdatedAt: updatedAt,
			},
		},
		{
			name:      "コンボランプなしは補正なしで計算しランプをnilにする",
			chart:     ChartRankingChart{Const: &chartConst},
			comboLamp: "NONE",
			expected: ChartRankingEntry{
				Rank: 2, Username: "user", PlayerName: "PLAYER", Score: 1_008_500,
				Rating: 16.6, Overpower: 84.0, OverpowerPercent: 96.0,
				ClearLamp: &clearLamp, UpdatedAt: updatedAt,
			},
		},
		{
			name:      "WORLD'S ENDは譜面定数があってもレーティングとOPを0にする",
			chart:     ChartRankingChart{Const: &chartConst, IsWorldsend: true},
			comboLamp: "ALL JUSTICE",
			expected: ChartRankingEntry{
				Rank: 2, Username: "user", PlayerName: "PLAYER", Score: 1_008_500,
				ClearLamp: &clearLamp, ComboLamp: &allJustice, UpdatedAt: updatedAt,
			},
		},
		{
			name:      "譜面定数が未設定の通常譜面はレーティングとOPを0にする",
			chart:     ChartRankingChart{IsConstUnknown: true},
			comboLamp: "ALL JUSTICE",
			expected: ChartRankingEntry{
				Rank: 2, Username: "user", PlayerName: "PLAYER", Score: 1_008_500,
				ClearLamp: &clearLamp, ComboLamp: &allJustice, UpdatedAt: updatedAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			record := &repository.ChartRankingRecord{
				Username:   "user",
				PlayerName: "PLAYER",
				Score:      1_008_500,
				ClearLamp:  "CLEAR",
				ComboLamp:  tt.comboLamp,
				FullChain:  "NONE",
				UpdatedAt:  updatedAt,
			}

			// When
			got := newChartRankingEntry(tt.chart, 2, record)

			// Then
			assert.Equal(t, tt.expected, got)
		})
	}
}
