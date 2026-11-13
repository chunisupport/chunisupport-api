package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyScores_成績不変なら履歴を作らず枠の差分だけを保存する(t *testing.T) {
	for _, tt := range []struct {
		name            string
		beforeSlot      int
		wantSlotUpdates int
	}{
		{"同じ枠は書き戻さない", 2, 1},
		{"枠の変更も反映する", 3, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 入力外の102と、入力中の101がどちらも枠付きの状態。
			repo := &stubPlayerDataRepositoryForApplyScoresTest{
				fullBefore:      map[int]repository.PlayerRecordState{101: {Score: 1010000, ClearLampID: 5, ComboLampID: 3, FullChainID: 2, SlotID: tt.beforeSlot, SlotOrder: new(1)}},
				worldsendBefore: map[int]repository.WorldsendRecordState{201: {Score: 1009000, ClearLampID: 3, ComboLampID: 2, FullChainID: 3}},
				rankedChartIDs:  []int{101, 102},
			}
			histories := &stubScoreHistoryRepositoryForApplyScoresTest{}
			uc := &playerDataUsecase{playerDataRepo: repo, playerRecRepo: &stubPlayerRecordRepositoryForApplyScoresTest{}, worldsendRecRepo: &stubWorldsendRecordRepositoryForApplyScoresTest{}, scoreHistoryRepo: histories}
			payload := PlayerDataScorePayload{
				Standard:  []PlayerDataScoreEntry{{Idx: "full-song", Diff: "MAS", Score: 1010000, ClearLamp: new("absolute"), ComboLv: new(3), FullChain: new(3), Slot: new("best"), Order: new(1)}},
				Worldsend: []PlayerDataScoreEntry{{Idx: "world-song", Score: 1009000, ClearLamp: new("hard"), ComboLv: new(2), FullChain: new(2)}},
			}
			// When
			counts, skipped, changes, _, _, err := uc.applyScores(context.Background(), nil, 77, payload, newApplyScoresTestMasters(), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), service.PlayerRecordStatisticsSnapshot{})
			// Then
			require.NoError(t, err)
			assert.Empty(t, skipped)
			assert.Empty(t, changes)
			assert.Equal(t, 1, counts.FullRecordsUpserted)
			assert.Equal(t, 1, counts.WorldsendRecordsUpserted)
			assert.Zero(t, counts.FullRecordsActuallyChanged)
			assert.Zero(t, counts.WorldsendRecordsActuallyChanged)
			assert.Empty(t, histories.standardRows)
			assert.Empty(t, histories.worldsendRows)
			assert.Empty(t, repo.savedInput.FullRecordsToInsert)
			assert.Empty(t, repo.savedInput.FullRecordsToUpdate)
			assert.Empty(t, repo.savedInput.WorldsendRecordsToInsert)
			assert.Empty(t, repo.savedInput.WorldsendRecordsToUpdate)
			require.Len(t, repo.savedInput.FullRecordSlotsToUpdate, tt.wantSlotUpdates)
			assert.Contains(t, repo.savedInput.FullRecordSlotsToUpdate, repository.PlayerRecordSlotUpdate{PlayerID: 77, ChartID: 102, SlotID: 1})
			if tt.beforeSlot != 2 {
				assert.Contains(t, repo.savedInput.FullRecordSlotsToUpdate, repository.PlayerRecordSlotUpdate{PlayerID: 77, ChartID: 101, SlotID: 2, SlotOrder: new(1)})
			}
		})
	}
}
