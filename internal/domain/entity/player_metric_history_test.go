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
		beforePercent   *float64
		afterRating     float64
		afterOverpower  float64
		afterPercent    float64
		expected        bool
	}{
		{name: "同じ値なら変更なし", beforeRating: 17.25, beforeOverpower: 12345.67, beforePercent: new(98.76), afterRating: 17.25, afterOverpower: 12345.67, afterPercent: 98.76, expected: false},
		{name: "RATINGが変われば変更あり", beforeRating: 17.24, beforeOverpower: 12345.67, beforePercent: new(98.76), afterRating: 17.25, afterOverpower: 12345.67, afterPercent: 98.76, expected: true},
		{name: "OPが変われば変更あり", beforeRating: 17.25, beforeOverpower: 12345.67, beforePercent: new(98.76), afterRating: 17.25, afterOverpower: 12346.01, afterPercent: 98.76, expected: true},
		{name: "OPパーセントが変われば変更あり", beforeRating: 17.25, beforeOverpower: 12345.67, beforePercent: new(98.75), afterRating: 17.25, afterOverpower: 12345.67, afterPercent: 98.76, expected: true},
		{name: "OPパーセントが未記録なら変更あり", beforeRating: 17.25, beforeOverpower: 12345.67, afterRating: 17.25, afterOverpower: 12345.67, afterPercent: 98.76, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &Player{OfficialRating: tt.beforeRating, OfficialOverpower: tt.beforeOverpower, OfficialOverpowerPercent: tt.beforePercent}
			actual := player.HasOfficialMetricsChanged(tt.afterRating, tt.afterOverpower, tt.afterPercent)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestPlayer_ChangeOfficialMetrics(t *testing.T) {
	collectedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name               string
		initialCollectedAt *time.Time
		inputAt            time.Time
		rating             float64
		overpower          float64
		overpowerPercent   float64
		wantErr            error
		wantHistory        bool
	}{
		{name: "新しい取得値へ更新すると変更前を履歴化する", initialCollectedAt: &collectedAt, inputAt: collectedAt.Add(time.Hour), rating: 17.25, overpower: 12345.67, overpowerPercent: 98.76, wantHistory: true},
		{name: "OPパーセントだけ変わっても変更前を履歴化する", initialCollectedAt: &collectedAt, inputAt: collectedAt.Add(time.Hour), rating: 17.24, overpower: 12340.12, overpowerPercent: 98.76, wantHistory: true},
		{name: "古い取得日時は拒否する", initialCollectedAt: &collectedAt, inputAt: collectedAt.Add(-time.Second), rating: 17.25, overpower: 12345.67, overpowerPercent: 98.76, wantErr: ErrStalePlayerData},
		{name: "同一取得日時の異なる値は拒否する", initialCollectedAt: &collectedAt, inputAt: collectedAt, rating: 17.25, overpower: 12345.67, overpowerPercent: 98.76, wantErr: ErrConflictingPlayerDataTimestamp},
		{name: "同一取得日時の同じ値は許可する", initialCollectedAt: &collectedAt, inputAt: collectedAt, rating: 17.24, overpower: 12340.12, overpowerPercent: 98.75},
		{name: "初回取得では履歴を作らず入力を反映する", inputAt: collectedAt.Add(time.Hour), rating: 17.25, overpower: 12345.67, overpowerPercent: 98.76},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 期待日時と入力の参照を分離し、破壊的な更新も検知できるようにします。
			var initialCollectedAt *time.Time
			if tt.initialCollectedAt != nil {
				initialCollectedAt = new(*tt.initialCollectedAt)
			}
			player := &Player{ID: 10, OfficialRating: 17.24, OfficialOverpower: 12340.12, OfficialOverpowerPercent: new(98.75), DataCollectedAt: initialCollectedAt, UpdatedAt: updatedAt}

			err := player.ChangeOfficialMetrics(tt.rating, tt.overpower, tt.overpowerPercent, tt.inputAt)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, 17.24, player.OfficialRating)
				assert.Equal(t, 12340.12, player.OfficialOverpower)
				require.NotNil(t, player.OfficialOverpowerPercent)
				assert.Equal(t, 98.75, *player.OfficialOverpowerPercent)
				require.NotNil(t, player.DataCollectedAt)
				assert.Equal(t, collectedAt, *player.DataCollectedAt)
				assert.Equal(t, updatedAt, player.UpdatedAt)
				assert.Nil(t, player.PendingMetricHistory())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.rating, player.OfficialRating)
			assert.Equal(t, tt.overpower, player.OfficialOverpower)
			require.NotNil(t, player.OfficialOverpowerPercent)
			assert.Equal(t, tt.overpowerPercent, *player.OfficialOverpowerPercent)
			require.NotNil(t, player.DataCollectedAt)
			assert.Equal(t, tt.inputAt, *player.DataCollectedAt)
			assert.Equal(t, tt.inputAt, player.UpdatedAt)
			if tt.wantHistory {
				require.NotNil(t, player.PendingMetricHistory())
				assert.Equal(t, &PlayerMetricHistoryEntry{PlayerID: 10, OfficialRating: 17.24, OfficialOverpower: 12340.12, OfficialOverpowerPercent: new(98.75), DataCollectedAt: collectedAt}, player.PendingMetricHistory())
			} else {
				assert.Nil(t, player.PendingMetricHistory())
			}
		})
	}
}
