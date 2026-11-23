package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	mastervo "github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyScores_本枠を保存できなければ枠解除前に拒否する(t *testing.T) {
	tests := []struct {
		name  string
		entry PlayerDataScoreEntry
	}{
		{name: "未登録の新曲", entry: PlayerDataScoreEntry{Idx: "missing-song", Diff: "MAS", Score: 1000000}},
		{name: "未登録の譜面", entry: PlayerDataScoreEntry{Idx: "full-song", Diff: "ULT", Score: 1000000}},
		{name: "範囲外スコア", entry: PlayerDataScoreEntry{Idx: "full-song", Diff: "MAS", Score: 1010001}},
		{name: "解決できないランプ", entry: PlayerDataScoreEntry{Idx: "full-song", Diff: "MAS", Score: 1000000, ClearLamp: new("unknown")}},
	}
	for _, slot := range []string{"best", " NEW "} {
		for _, tt := range tests {
			t.Run(slot+"/"+tt.name, func(t *testing.T) {
				// Given: 末尾の本枠が保存できない場合も、欠番の検証だけで見逃さない。
				entry := tt.entry
				entry.Slot = &slot
				entry.Order = new(2)
				masters := newApplyScoresTestMasters()
				masters.Difficulties["ULTIMA"] = mastervo.ChartDifficulty{ID: 5, Name: "ULTIMA"}
				masters.songs["existing-song"] = entity.PlayerDataSong{ID: 3, OfficialIdx: "existing-song"}
				masters.chartsByKey["3:4"] = entity.PlayerDataChart{ID: 102, SongID: 3, DifficultyID: 4}
				payload := PlayerDataScorePayload{Standard: []PlayerDataScoreEntry{
					{Idx: "existing-song", Diff: "MAS", Score: 1000000, Slot: &slot, Order: new(1)},
					entry,
				}}
				repo := &stubPlayerDataRepositoryForApplyScoresTest{}
				uc := &playerDataUsecase{playerDataRepo: repo, playerRecRepo: &stubPlayerRecordRepositoryForApplyScoresTest{}, worldsendRecRepo: &stubWorldsendRecordRepositoryForApplyScoresTest{}}

				// When
				_, _, _, _, _, err := uc.applyScores(context.Background(), nil, 77, payload, masters, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), service.PlayerRecordStatisticsSnapshot{})

				// Then
				var validationErr *PlayerDataValidationError
				require.ErrorAs(t, err, &validationErr)
				assert.Equal(t, "scores.standard", validationErr.Field)
				assert.Zero(t, repo.findRankedRecordsCalls)
				assert.Zero(t, repo.saveCalls)
			})
		}
	}
}

func TestValidateFullRecordSlots_本枠は1位から連続している必要がある(t *testing.T) {
	tests := []struct {
		name    string
		orders  []int
		wantErr bool
	}{
		{name: "空枠", orders: nil},
		{name: "1曲だけ", orders: []int{1}},
		{name: "上限未満で連続", orders: []int{3, 1, 2}},
		{name: "先頭の難易度が欠落", orders: []int{2, 3}, wantErr: true},
		{name: "途中の難易度が欠落", orders: []int{1, 3}, wantErr: true},
	}
	for _, slotID := range []int{2, 3} {
		for _, tt := range tests {
			t.Run(newApplyScoresTestMasters().SlotNamesByID[slotID]+"/"+tt.name, func(t *testing.T) {
				// Given
				records := make([]repository.PlayerRecordForUpsert, 0, len(tt.orders))
				for i, order := range tt.orders {
					records = append(records, repository.PlayerRecordForUpsert{ChartID: i + 1, State: repository.PlayerRecordState{SlotID: slotID, SlotOrder: &order}})
				}

				// When
				err := validateFullRecordSlots(records, newApplyScoresTestMasters())

				// Then
				if tt.wantErr {
					var validationErr *PlayerDataValidationError
					require.ErrorAs(t, err, &validationErr)
					assert.Equal(t, "scores.standard", validationErr.Field)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	}
}

func TestFindCandidatesAboveMainSlot_本枠下限を上回る候補枠を枠ごとに返す(t *testing.T) {
	// Given: ベスト枠と新曲枠の下限はどちらも定数15.0の1,000,000点（単曲レート15.00）。
	masters := newApplyScoresTestMasters()
	for id := 1; id <= 6; id++ {
		masters.chartsByID[id] = entity.PlayerDataChart{ID: id, Const: chartconstant.ChartConstant(15.0)}
	}
	masters.chartsByID[7] = entity.PlayerDataChart{ID: 7, Const: chartconstant.ChartConstant(15.0), IsConstUnknown: true}
	record := func(chartID, slotID, order, score int) repository.PlayerRecordForUpsert {
		return repository.PlayerRecordForUpsert{ChartID: chartID, State: repository.PlayerRecordState{Score: score, SlotID: slotID, SlotOrder: &order}}
	}
	records := []repository.PlayerRecordForUpsert{
		record(1, 2, 1, 1_000_000),
		record(2, 3, 1, 1_000_000),
		record(3, 4, 1, 1_005_000), // ベスト候補: 下限超過
		record(4, 4, 2, 990_000),   // ベスト候補: 正常
		record(5, 5, 1, 1_000_000), // 新曲候補: 同値は正常
		record(6, 5, 2, 1_007_500), // 新曲候補: 下限超過
		record(7, 5, 3, 1_009_000), // 新曲候補: 定数不明のため比較しない
	}

	// When
	result := findCandidatesAboveMainSlot(records, masters)

	// Then
	assert.Equal(t, map[string][]int{"best_candidate": {3}, "new_candidate": {6}}, result)
}

func TestFindCandidatesAboveMainSlot_定数不明の本枠は下限の計算に使わない(t *testing.T) {
	// Given: 定数不明の本枠を下限に含めると、正常な候補枠を誤って検出してしまう。
	masters := newApplyScoresTestMasters()
	masters.chartsByID[1] = entity.PlayerDataChart{ID: 1, Const: chartconstant.ChartConstant(15.0)}
	masters.chartsByID[2] = entity.PlayerDataChart{ID: 2, Const: chartconstant.ChartConstant(1.0), IsConstUnknown: true}
	masters.chartsByID[3] = entity.PlayerDataChart{ID: 3, Const: chartconstant.ChartConstant(14.0)}
	order := 1
	records := []repository.PlayerRecordForUpsert{
		{ChartID: 1, State: repository.PlayerRecordState{Score: 1_000_000, SlotID: 2, SlotOrder: &order}},
		{ChartID: 2, State: repository.PlayerRecordState{Score: 1_000_000, SlotID: 2, SlotOrder: new(2)}},
		{ChartID: 3, State: repository.PlayerRecordState{Score: 1_000_000, SlotID: 4, SlotOrder: &order}},
	}

	// When
	result := findCandidatesAboveMainSlot(records, masters)

	// Then
	assert.Empty(t, result)
}
