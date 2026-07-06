package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerStatsRecalculationBatchUsecase_運用日を07時境界で固定する(t *testing.T) {
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
			usecase := NewPlayerStatsRecalculationBatchUsecase(repo)
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

	update, err := prepared.buildUpdate(repository.PlayerBatchData{
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

func validBatchSnapshot() repository.PlayerStatsMasterSnapshot {
	return repository.PlayerStatsMasterSnapshot{
		Version: repository.BatchVersion{ID: 1, Name: "VERSE", ReleasedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		SlotIDs: map[string]int{"none": 1, "best": 2, "best_candidate": 3, "new": 4, "new_candidate": 5},
	}
}

type batchRepositoryStub struct {
	snapshot        repository.PlayerStatsMasterSnapshot
	operationalDate time.Time
}

func (s *batchRepositoryStub) LoadSnapshot(_ context.Context, operationalDate time.Time) (repository.PlayerStatsMasterSnapshot, error) {
	s.operationalDate = operationalDate
	return s.snapshot, nil
}

func (s *batchRepositoryStub) ListPlayerKeys(context.Context, int, int, int) ([]repository.PlayerBatchKey, error) {
	return nil, nil
}

func (s *batchRepositoryStub) ProcessPlayer(context.Context, repository.PlayerBatchKey, func(repository.PlayerBatchData) (repository.PlayerBatchUpdate, error)) (repository.PlayerBatchProcessStatus, error) {
	return repository.PlayerBatchUpdated, nil
}
