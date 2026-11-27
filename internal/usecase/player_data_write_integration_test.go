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

func TestApplyScores_同スコアでも虹FULLCHAINへの変化を保存する(t *testing.T) {
	before := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	after := before.Add(time.Hour)
	repo := &stubPlayerDataRepositoryForApplyScoresTest{
		fullBefore: map[int]repository.PlayerRecordState{
			101: {Score: 1010000, ClearLampID: 5, ComboLampID: 3, FullChainID: 3, SlotID: 2, SlotOrder: new(1), UpdatedAt: before},
		},
		worldsendBefore: map[int]repository.WorldsendRecordState{
			201: {Score: 1009000, ClearLampID: 3, ComboLampID: 3, FullChainID: 3, UpdatedAt: before},
		},
	}
	histories := &stubScoreHistoryRepositoryForApplyScoresTest{}
	uc := &playerDataUsecase{
		playerDataRepo:   repo,
		playerRecRepo:    &stubPlayerRecordRepositoryForApplyScoresTest{},
		worldsendRecRepo: &stubWorldsendRecordRepositoryForApplyScoresTest{},
		scoreHistoryRepo: histories,
	}
	masters := newApplyScoresTestMasters()
	masters.DifficultyNamesByID = map[int]string{4: "MASTER"}
	platinum := masters.FullChains["full chain platinum"]
	platinum.Name = "FULL CHAIN PLATINUM"
	masters.FullChains["full chain platinum"] = platinum
	payload := PlayerDataScorePayload{
		Standard:  []PlayerDataScoreEntry{{Idx: "full-song", Diff: "MAS", Score: 1010000, ClearLamp: new("absolute"), ComboLv: new(3), FullChain: new(4), Slot: new("best"), Order: new(1)}},
		Worldsend: []PlayerDataScoreEntry{{Idx: "world-song", Score: 1009000, ClearLamp: new("hard"), ComboLv: new(3), FullChain: new(4)}},
	}

	counts, skipped, changes, _, _, err := uc.applyScores(context.Background(), nil, 77, payload, masters, after, service.PlayerRecordStatisticsSnapshot{})

	require.NoError(t, err)
	assert.Empty(t, skipped)
	assert.Equal(t, 1, counts.FullRecordsActuallyChanged)
	assert.Equal(t, 1, counts.WorldsendRecordsActuallyChanged)
	require.Len(t, changes, 2)
	for _, change := range changes {
		require.NotNil(t, change.Before)
		assert.Equal(t, change.Before.Score, change.After.Score)
		assert.Equal(t, new("FULL CHAIN PLATINUM"), change.Before.FullChain)
		assert.Equal(t, new("FULL CHAIN RAINBOW"), change.After.FullChain)
	}
	require.Len(t, repo.savedInput.FullRecordsToUpdate, 1)
	require.Len(t, repo.savedInput.WorldsendRecordsToUpdate, 1)
	assert.Equal(t, 17, repo.savedInput.FullRecordsToUpdate[0].State.FullChainID)
	assert.Equal(t, 17, repo.savedInput.WorldsendRecordsToUpdate[0].State.FullChainID)
	require.Len(t, histories.standardRows, 1)
	require.Len(t, histories.worldsendRows, 1)
	assert.Equal(t, 3, histories.standardRows[0].State.FullChainID)
	assert.Equal(t, 3, histories.worldsendRows[0].State.FullChainID)
}

func TestBuildPlayLog_虹FULLCHAINを登録する(t *testing.T) {
	entry := validPlayLogEntry()
	entry.ComboLv = new(3)
	entry.FullChain = new(4)
	masters := newApplyScoresTestMasters()

	log, err := buildPlayLog(0, entry, masters)
	require.NoError(t, err)
	assert.Equal(t, 17, log.FullChainID)

	entry.ComboLv = new(2)
	_, err = buildPlayLog(0, entry, masters)
	assert.ErrorContains(t, err, "without AJ")
}
