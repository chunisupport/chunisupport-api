package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerDataRecalculationBatchUsecase_運用日を07時境界で固定する(t *testing.T) {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	tests := []struct {
		name     string
		now      time.Time
		expected string
	}{
		{name: "06時59分は前日", now: time.Date(2026, 7, 6, 6, 59, 0, 0, jst), expected: "2026-07-05"},
		{name: "07時00分は当日", now: time.Date(2026, 7, 6, 7, 0, 0, 0, jst), expected: "2026-07-06"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &batchRepositoryStub{snapshot: validBatchSnapshot()}
			usecase := NewPlayerDataRecalculationBatchUsecase(repo)
			usecase.now = func() time.Time { return tt.now }

			_, err := usecase.Execute(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.expected, repo.operationalDate.Format(time.DateOnly))
		})
	}
}

func TestPreparedBatchSnapshot_旧版の新曲とベストを排他的に再構築する(t *testing.T) {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	snapshot := validBatchSnapshot()
	oldDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	currentDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	snapshot.Songs = []repository.BatchSong{
		{ID: 1, ReleasedAt: &oldDate, OfficialIndex: "1"},
		{ID: 2, ReleasedAt: &currentDate, OfficialIndex: "2"},
		{ID: 3, ReleasedAt: nil, OfficialIndex: "3"},
	}
	snapshot.Charts = []repository.BatchChart{
		{ID: 1, SongID: 1, DifficultyName: "MASTER", ChartConst: 15},
		{ID: 2, SongID: 2, DifficultyName: "MASTER", ChartConst: 15},
		{ID: 3, SongID: 3, DifficultyName: "MASTER", ChartConst: 15},
	}
	prepared, err := prepareBatchSnapshot(snapshot, time.Date(2026, 7, 6, 0, 0, 0, 0, jst), jst)
	require.NoError(t, err)

	update, _, err := prepared.buildUpdate(repository.PlayerBatchData{
		ID: 1,
		Records: []repository.PlayerBatchRecord{
			{ChartID: 1, Score: 1_009_000},
			{ChartID: 2, Score: 1_009_000},
			{ChartID: 3, Score: 1_009_000},
		},
	}, false)

	require.NoError(t, err)
	require.Len(t, update.Assignments, 3)
	assert.Equal(t, []int{snapshot.SlotIDs["best"], snapshot.SlotIDs["new"], snapshot.SlotIDs["new"]},
		[]int{update.Assignments[0].SlotID, update.Assignments[1].SlotID, update.Assignments[2].SlotID})
}

func TestPreparedBatchSnapshot_現行の正常な公式枠を保持する(t *testing.T) {
	// Given
	prepared := preparedBatchSnapshotForSlotTest(t, 2)
	data := repository.PlayerBatchData{ID: 1, Records: []repository.PlayerBatchRecord{
		{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
		{ChartID: 2, Score: 1_008_000, SlotName: "new", SlotOrder: batchIntPtr(1)},
	}}

	// When
	update, currentBroken, err := prepared.buildUpdate(data, true)

	// Then
	require.NoError(t, err)
	assert.False(t, currentBroken)
	assert.False(t, update.ResetSlots)
	assert.Empty(t, update.Assignments)
}

func TestPreparedBatchSnapshot_現行の壊れた本枠を再構築する(t *testing.T) {
	tests := []struct {
		name    string
		records []repository.PlayerBatchRecord
	}{
		{
			name: "bestの順位重複",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
				{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
			},
		},
		{
			name: "newの順位未設定",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "new"},
			},
		},
		{
			name: "bestの順位範囲外",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(31)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			prepared := preparedBatchSnapshotForSlotTest(t, 2)

			// When
			update, currentBroken, err := prepared.buildUpdate(repository.PlayerBatchData{ID: 1, Records: tt.records}, true)

			// Then
			require.NoError(t, err)
			assert.True(t, currentBroken)
			assert.True(t, update.ResetSlots)
			assert.NotEmpty(t, update.Assignments)
		})
	}
}

func TestPreparedBatchSnapshot_現行の候補枠だけが壊れていても保持する(t *testing.T) {
	// Given
	prepared := preparedBatchSnapshotForSlotTest(t, 3)
	data := repository.PlayerBatchData{ID: 1, Records: []repository.PlayerBatchRecord{
		{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
		{ChartID: 2, Score: 1_008_000, SlotName: "best_candidate", SlotOrder: batchIntPtr(1)},
		{ChartID: 3, Score: 1_007_000, SlotName: "best_candidate", SlotOrder: batchIntPtr(1)},
	}}

	// When
	update, currentBroken, err := prepared.buildUpdate(data, true)

	// Then
	require.NoError(t, err)
	assert.False(t, currentBroken)
	assert.False(t, update.ResetSlots)
}

func TestPreparedBatchSnapshot_現行本枠の順位が連番でなくても保持する(t *testing.T) {
	// Given
	prepared := preparedBatchSnapshotForSlotTest(t, 2)
	data := repository.PlayerBatchData{ID: 1, Records: []repository.PlayerBatchRecord{
		{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
		{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(3)},
	}}

	// When
	update, currentBroken, err := prepared.buildUpdate(data, true)

	// Then
	require.NoError(t, err)
	assert.False(t, currentBroken)
	assert.False(t, update.ResetSlots)
}

func TestPreparedBatchSnapshot_現行の壊れた本枠をリリース日で再分類する(t *testing.T) {
	// Given
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	snapshot := batchSnapshotForSlotTest(2)
	currentDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	snapshot.Songs[1].ReleasedAt = &currentDate
	prepared, err := prepareBatchSnapshot(snapshot, time.Date(2026, 7, 6, 0, 0, 0, 0, jst), jst)
	require.NoError(t, err)
	data := repository.PlayerBatchData{ID: 1, Records: []repository.PlayerBatchRecord{
		{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
		{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
	}}

	// When
	update, currentBroken, err := prepared.buildUpdate(data, true)

	// Then
	require.NoError(t, err)
	assert.True(t, currentBroken)
	require.Len(t, update.Assignments, 2)
	assert.Equal(t, []int{snapshot.SlotIDs["best"], snapshot.SlotIDs["new"]},
		[]int{update.Assignments[0].SlotID, update.Assignments[1].SlotID})
}

func TestPreparedBatchSnapshot_現行の壊れた本枠から候補枠と指標も再計算する(t *testing.T) {
	// Given
	snapshot := batchSnapshotForSlotTest(31)
	snapshot.Charts[30].ChartConst = 16
	prepared := preparedBatchSnapshotForCustomSnapshot(t, snapshot)
	records := make([]repository.PlayerBatchRecord, 0, 31)
	for i := 1; i <= 30; i++ {
		record := repository.PlayerBatchRecord{ChartID: i, Score: 1_009_000}
		if i <= 2 {
			record.SlotName = "best"
			record.SlotOrder = batchIntPtr(1)
		}
		records = append(records, record)
	}
	records = append(records, repository.PlayerBatchRecord{ChartID: 31, Score: 1_000_000})

	// When
	update, currentBroken, err := prepared.buildUpdate(repository.PlayerBatchData{ID: 1, Records: records}, true)

	// Then
	require.NoError(t, err)
	assert.True(t, currentBroken)
	assert.True(t, update.ResetSlots)
	require.Len(t, update.Assignments, 31)
	assert.Equal(t, snapshot.SlotIDs["best_candidate"], update.Assignments[30].SlotID)
	assert.Positive(t, update.PlayerRating)
	assert.Positive(t, update.BestAverage)
	assert.Positive(t, update.Overpower)
}

func TestValidateOfficialSlotSet_本枠の件数超過を検出する(t *testing.T) {
	// Given
	records := make([]repository.PlayerBatchRecord, 0, 31)
	for i := 1; i <= 31; i++ {
		records = append(records, repository.PlayerBatchRecord{ChartID: i, SlotName: "best", SlotOrder: batchIntPtr(min(i, 30))})
	}

	// When
	err := validateOfficialSlotSet(records, "best", 30)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "best枠が30件を超えています")
}

func TestPlayerDataRecalculationBatchUsecase_現行の壊れた本枠を成功として数える(t *testing.T) {
	// Given
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	snapshot := batchSnapshotForSlotTest(2)
	lastPlayedAt := time.Date(2026, 7, 1, 7, 0, 0, 0, jst)
	repo := &batchRepositoryStub{
		snapshot: snapshot,
		keys:     []repository.PlayerBatchKey{{ID: 1}},
		data: map[int]repository.PlayerBatchData{1: {ID: 1, LastPlayedAt: &lastPlayedAt, Records: []repository.PlayerBatchRecord{
			{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
			{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
		}}},
	}
	usecase := NewPlayerDataRecalculationBatchUsecase(repo)
	usecase.now = func() time.Time { return time.Date(2026, 7, 6, 12, 0, 0, 0, jst) }

	// When
	result, err := usecase.Execute(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, 1, result.Success)
	assert.Equal(t, 1, result.CurrentBrokenRebuilt)
	assert.Zero(t, result.CurrentPreserved)
	assert.Zero(t, result.Failed)
	require.True(t, repo.updates[1].ResetSlots)
}

func TestPlayerDataRecalculationBatchUsecase_壊れた本枠の保存失敗を再構築成功として数えない(t *testing.T) {
	// Given
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	lastPlayedAt := time.Date(2026, 7, 1, 7, 0, 0, 0, jst)
	repo := &batchRepositoryStub{
		snapshot:      batchSnapshotForSlotTest(2),
		keys:          []repository.PlayerBatchKey{{ID: 1}},
		afterBuildErr: assert.AnError,
		data: map[int]repository.PlayerBatchData{1: {ID: 1, LastPlayedAt: &lastPlayedAt, Records: []repository.PlayerBatchRecord{
			{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
			{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
		}}},
	}
	usecase := NewPlayerDataRecalculationBatchUsecase(repo)
	usecase.now = func() time.Time { return time.Date(2026, 7, 6, 12, 0, 0, 0, jst) }

	// When
	result, err := usecase.Execute(context.Background())

	// Then
	require.Error(t, err)
	assert.Zero(t, result.Success)
	assert.Zero(t, result.CurrentBrokenRebuilt)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, "current_broken_slots", playerRebuildReason(true, true, &lastPlayedAt))
}

func TestPlayerDataRecalculationBatchUsecase_バージョン開始時刻で現行を判定する(t *testing.T) {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	tests := []struct {
		name                 string
		lastPlayedAt         time.Time
		wantCurrentPreserved int
		wantLegacyRebuilt    int
		wantResetSlots       bool
	}{
		{
			name:              "開始直前は旧版",
			lastPlayedAt:      time.Date(2026, 7, 1, 6, 59, 59, 0, jst),
			wantLegacyRebuilt: 1,
			wantResetSlots:    true,
		},
		{
			name:                 "開始時刻は現行",
			lastPlayedAt:         time.Date(2026, 7, 1, 7, 0, 0, 0, jst),
			wantCurrentPreserved: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &batchRepositoryStub{
				snapshot: validBatchSnapshot(),
				keys:     []repository.PlayerBatchKey{{ID: 1}},
				data:     map[int]repository.PlayerBatchData{1: {ID: 1, LastPlayedAt: &tt.lastPlayedAt}},
			}
			usecase := NewPlayerDataRecalculationBatchUsecase(repo)
			usecase.now = func() time.Time { return time.Date(2026, 7, 6, 12, 0, 0, 0, jst) }

			// When
			result, err := usecase.Execute(context.Background())

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wantCurrentPreserved, result.CurrentPreserved)
			assert.Equal(t, tt.wantLegacyRebuilt, result.LegacyRebuilt)
			assert.Equal(t, tt.wantResetSlots, repo.updates[1].ResetSlots)
		})
	}
}

func TestDatabaseDateInLocation_DATEのロケーションに依存しない(t *testing.T) {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	value := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "2026-07-06", databaseDateInLocation(value, jst).Format(time.DateOnly))
}

func TestPrepareBatchSnapshot_officialIdxを数値へ変換する(t *testing.T) {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	snapshot := validBatchSnapshot()
	snapshot.Songs = []repository.BatchSong{
		{ID: 1, OfficialIndex: "10"},
		{ID: 2, OfficialIndex: "002"},
	}

	prepared, err := prepareBatchSnapshot(snapshot, time.Date(2026, 7, 6, 0, 0, 0, 0, jst), jst)

	require.NoError(t, err)
	assert.Equal(t, uint64(10), prepared.officialIndex[1])
	assert.Equal(t, uint64(2), prepared.officialIndex[2])
}

func validBatchSnapshot() repository.PlayerDataMasterSnapshot {
	return repository.PlayerDataMasterSnapshot{
		Version: repository.BatchVersion{ID: 1, Name: "VERSE", ReleasedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		SlotIDs: map[string]int{"none": 1, "best": 2, "best_candidate": 3, "new": 4, "new_candidate": 5},
	}
}

type batchRepositoryStub struct {
	snapshot        repository.PlayerDataMasterSnapshot
	operationalDate time.Time
	keys            []repository.PlayerBatchKey
	data            map[int]repository.PlayerBatchData
	updates         map[int]repository.PlayerBatchUpdate
	afterBuildErr   error
}

func (s *batchRepositoryStub) LoadSnapshot(_ context.Context, operationalDate time.Time) (repository.PlayerDataMasterSnapshot, error) {
	s.operationalDate = operationalDate
	return s.snapshot, nil
}

func (s *batchRepositoryStub) ListPlayerKeys(_ context.Context, afterID, _ int, _ int) ([]repository.PlayerBatchKey, error) {
	if afterID > 0 {
		return nil, nil
	}
	return s.keys, nil
}

func (s *batchRepositoryStub) ProcessPlayer(_ context.Context, key repository.PlayerBatchKey, buildUpdate func(repository.PlayerBatchData) (repository.PlayerBatchUpdate, error)) (repository.PlayerBatchProcessStatus, error) {
	update, err := buildUpdate(s.data[key.ID])
	if err != nil {
		return repository.PlayerBatchUpdated, err
	}
	if s.updates == nil {
		s.updates = make(map[int]repository.PlayerBatchUpdate)
	}
	s.updates[key.ID] = update
	if s.afterBuildErr != nil {
		return repository.PlayerBatchUpdated, s.afterBuildErr
	}
	return repository.PlayerBatchUpdated, nil
}

func preparedBatchSnapshotForSlotTest(t *testing.T, recordCount int) preparedBatchSnapshot {
	t.Helper()
	return preparedBatchSnapshotForCustomSnapshot(t, batchSnapshotForSlotTest(recordCount))
}

func preparedBatchSnapshotForCustomSnapshot(t *testing.T, snapshot repository.PlayerDataMasterSnapshot) preparedBatchSnapshot {
	t.Helper()
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	prepared, err := prepareBatchSnapshot(snapshot, time.Date(2026, 7, 6, 0, 0, 0, 0, jst), jst)
	require.NoError(t, err)
	return prepared
}

func batchSnapshotForSlotTest(recordCount int) repository.PlayerDataMasterSnapshot {
	snapshot := validBatchSnapshot()
	oldDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= recordCount; i++ {
		snapshot.Songs = append(snapshot.Songs, repository.BatchSong{ID: i, ReleasedAt: &oldDate, OfficialIndex: fmt.Sprintf("%d", i)})
		snapshot.Charts = append(snapshot.Charts, repository.BatchChart{ID: i, SongID: i, DifficultyName: "MASTER", ChartConst: 15})
	}
	return snapshot
}

func batchIntPtr(value int) *int {
	return &value
}
