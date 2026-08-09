package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayer_HasOfficialMetricsChanged(t *testing.T) {
	tests := []struct {
		name            string
		beforeRating    float64
		beforeOverpower float64
		afterRating     float64
		afterOverpower  float64
		expected        bool
	}{
		{name: "同じ値なら変更なし", beforeRating: 17.25, beforeOverpower: 12345.67, afterRating: 17.25, afterOverpower: 12345.67, expected: false},
		{name: "RATINGが変われば変更あり", beforeRating: 17.24, beforeOverpower: 12345.67, afterRating: 17.25, afterOverpower: 12345.67, expected: true},
		{name: "OPが変われば変更あり", beforeRating: 17.25, beforeOverpower: 12345.67, afterRating: 17.25, afterOverpower: 12346.01, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &Player{OfficialRating: tt.beforeRating, OfficialOverpower: tt.beforeOverpower}
			actual := player.HasOfficialMetricsChanged(tt.afterRating, tt.afterOverpower)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestPlayer_ChangeOfficialMetrics(t *testing.T) {
	collectedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		inputAt     time.Time
		rating      float64
		overpower   float64
		wantErr     error
		wantHistory bool
	}{
		{name: "新しい取得値へ更新すると変更前を履歴化する", inputAt: collectedAt.Add(time.Hour), rating: 17.25, overpower: 12345.67, wantHistory: true},
		{name: "古い取得日時は拒否する", inputAt: collectedAt.Add(-time.Second), rating: 17.25, overpower: 12345.67, wantErr: ErrStalePlayerData},
		{name: "同一取得日時の異なる値は拒否する", inputAt: collectedAt, rating: 17.25, overpower: 12345.67, wantErr: ErrConflictingPlayerDataTimestamp},
		{name: "同一取得日時の同じ値は許可する", inputAt: collectedAt, rating: 17.24, overpower: 12340.12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &Player{ID: 10, OfficialRating: 17.24, OfficialOverpower: 12340.12, DataCollectedAt: &collectedAt}

			err := player.ChangeOfficialMetrics(tt.rating, tt.overpower, tt.inputAt)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantHistory {
				require.NotNil(t, player.PendingMetricHistory())
				assert.Equal(t, collectedAt, player.PendingMetricHistory().DataCollectedAt)
			} else {
				assert.Nil(t, player.PendingMetricHistory())
			}
		})
	}
}
