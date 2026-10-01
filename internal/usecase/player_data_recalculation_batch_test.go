package usecase

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
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

func TestPreparedBatchSnapshot_両本枠欠落の修復(t *testing.T) {
	tests := []struct {
		name       string
		slot       string
		excluded   string
		noRecords  bool
		wantBroken bool
	}{
		{name: "通常記録から両枠を復元", slot: "none", wantBroken: true},
		{name: "候補枠だけでも両枠を復元", slot: "best_candidate", wantBroken: true},
		{name: "記録なし", noRecords: true},
		{name: "削除済み楽曲のみ", slot: "none", excluded: "deleted"},
		{name: "WORLD'S ENDのみ", slot: "none", excluded: "worldsend"},
		{name: "配信前楽曲のみ", slot: "none", excluded: "future"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := batchSnapshotForSlotTest(2)
			currentDate := snapshot.Version.ReleasedAt
			snapshot.Songs[1].ReleasedAt = &currentDate
			for i := range snapshot.Songs {
				switch tt.excluded {
				case "deleted":
					snapshot.Songs[i].IsDeleted = true
				case "worldsend":
					snapshot.Songs[i].IsWorldsend = true
				case "future":
					date := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
					snapshot.Songs[i].ReleasedAt = &date
				}
			}
			prepared := preparedBatchSnapshotForCustomSnapshot(t, snapshot)
			data := repository.PlayerBatchData{ID: 1}
			if !tt.noRecords {
				data.Records = []repository.PlayerBatchRecord{
					{ChartID: 1, Score: 1_009_000, SlotName: tt.slot},
					{ChartID: 2, Score: 1_008_000, SlotName: "none"},
				}
			}

			update, broken, err := prepared.buildUpdate(data, true)

			require.NoError(t, err)
			assert.Equal(t, tt.wantBroken, broken)
			if tt.wantBroken {
				assert.Equal(t, []repository.PlayerBatchSlotAssignment{
					{ChartID: 1, SlotID: snapshot.SlotIDs["best"], Position: 1},
					{ChartID: 2, SlotID: snapshot.SlotIDs["new"], Position: 1},
				}, update.Assignments)
				assert.Equal(t, 17.15, update.BestAverage)
				assert.Equal(t, 17.05, update.NewAverage)
				assert.Equal(t, 0.684, update.PlayerRating)
			} else {
				assert.Empty(t, update.Assignments)
				assert.Empty(t, update.ClearChartIDs)
				assert.Zero(t, update.BestAverage)
				assert.Zero(t, update.NewAverage)
			}
		})
	}
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
	assert.Empty(t, update.ClearChartIDs)
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
	assert.Empty(t, update.ClearChartIDs)
	assert.Empty(t, update.Assignments)
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
	assert.Empty(t, update.ClearChartIDs)
	assert.Empty(t, update.Assignments)
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
	// 譜面1はbest1位のまま変わらないため、new枠へ移る譜面2だけを更新します。
	assert.Empty(t, update.ClearChartIDs)
	assert.Equal(t, []repository.PlayerBatchSlotAssignment{{ChartID: 2, SlotID: snapshot.SlotIDs["new"], Position: 1}}, update.Assignments)
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
	assert.Empty(t, update.ClearChartIDs)
	// 譜面1はbest1位のまま変わらないため、残る30件（本枠29件と候補枠1件）だけを更新します。
	require.Len(t, update.Assignments, 30)
	assert.Equal(t, repository.PlayerBatchSlotAssignment{ChartID: 31, SlotID: snapshot.SlotIDs["best_candidate"], Position: 1}, update.Assignments[29])
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
	require.NotEmpty(t, repo.updates[1].Assignments)
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
		wantSlotsUnchanged   int
	}{
		{
			name:               "開始直前は旧版",
			lastPlayedAt:       time.Date(2026, 7, 1, 6, 59, 59, 0, jst),
			wantLegacyRebuilt:  1,
			wantSlotsUnchanged: 1,
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
			assert.Equal(t, tt.wantSlotsUnchanged, result.SlotsUnchanged)
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

func TestComputeMasterFingerprint_同じマスタからは並び順によらず同じ値になる(t *testing.T) {
	// Given
	snapshot := batchSnapshotForSlotTest(3)
	reordered := batchSnapshotForSlotTest(3)
	slices.Reverse(reordered.Songs)
	slices.Reverse(reordered.Charts)

	// When
	first, err := computeMasterFingerprint(snapshot)
	require.NoError(t, err)
	second, err := computeMasterFingerprint(reordered)
	require.NoError(t, err)

	// Then
	assert.Equal(t, first, second)
}

func TestComputeMasterFingerprint_計算に使う項目の変更で値が変わる(t *testing.T) {
	otherDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		change func(s *repository.PlayerDataMasterSnapshot)
	}{
		{name: "バージョンID", change: func(s *repository.PlayerDataMasterSnapshot) { s.Version.ID = 2 }},
		{name: "バージョン開始日", change: func(s *repository.PlayerDataMasterSnapshot) { s.Version.ReleasedAt = otherDate }},
		{name: "楽曲の追加", change: func(s *repository.PlayerDataMasterSnapshot) {
			s.Songs = append(s.Songs, repository.BatchSong{ID: 99, OfficialIndex: "99"})
		}},
		{name: "楽曲の配信日", change: func(s *repository.PlayerDataMasterSnapshot) { s.Songs[0].ReleasedAt = &otherDate }},
		{name: "楽曲の配信日がNULL", change: func(s *repository.PlayerDataMasterSnapshot) { s.Songs[0].ReleasedAt = nil }},
		{name: "楽曲の削除", change: func(s *repository.PlayerDataMasterSnapshot) { s.Songs[0].IsDeleted = true }},
		{name: "WORLD'S END", change: func(s *repository.PlayerDataMasterSnapshot) { s.Songs[0].IsWorldsend = true }},
		{name: "official_idx", change: func(s *repository.PlayerDataMasterSnapshot) { s.Songs[0].OfficialIndex = "100" }},
		{name: "譜面の追加", change: func(s *repository.PlayerDataMasterSnapshot) {
			s.Charts = append(s.Charts, repository.BatchChart{ID: 99, SongID: 1, DifficultyName: "EXPERT", ChartConst: 13})
		}},
		{name: "譜面の楽曲", change: func(s *repository.PlayerDataMasterSnapshot) { s.Charts[0].SongID = 2 }},
		{name: "譜面の難易度名", change: func(s *repository.PlayerDataMasterSnapshot) { s.Charts[0].DifficultyName = "ULTIMA" }},
		{name: "譜面定数", change: func(s *repository.PlayerDataMasterSnapshot) { s.Charts[0].ChartConst = 15.1 }},
		{name: "枠ID", change: func(s *repository.PlayerDataMasterSnapshot) {
			s.SlotIDs = map[string]int{"none": 1, "best": 3, "best_candidate": 2, "new": 4, "new_candidate": 5}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			base, err := computeMasterFingerprint(batchSnapshotForSlotTest(3))
			require.NoError(t, err)
			changed := batchSnapshotForSlotTest(3)
			tt.change(&changed)

			// When
			got, err := computeMasterFingerprint(changed)

			// Then
			require.NoError(t, err)
			assert.NotEqual(t, base, got)
		})
	}
}

func TestComputeMasterFingerprint_計算に使わない項目の変更では値が変わらない(t *testing.T) {
	tests := []struct {
		name   string
		change func(s *repository.PlayerDataMasterSnapshot)
	}{
		{name: "バージョン名", change: func(s *repository.PlayerDataMasterSnapshot) { s.Version.Name = "X-VERSE" }},
		{name: "譜面定数不明フラグ", change: func(s *repository.PlayerDataMasterSnapshot) { s.Charts[0].IsConstUnknown = true }},
		{name: "難易度ID", change: func(s *repository.PlayerDataMasterSnapshot) { s.Charts[0].DifficultyID = 9 }},
		{name: "プレイヤーIDの上限", change: func(s *repository.PlayerDataMasterSnapshot) { s.UpperBound = 1000 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			base, err := computeMasterFingerprint(batchSnapshotForSlotTest(3))
			require.NoError(t, err)
			changed := batchSnapshotForSlotTest(3)
			tt.change(&changed)

			// When
			got, err := computeMasterFingerprint(changed)

			// Then
			require.NoError(t, err)
			assert.Equal(t, base, got)
		})
	}
}

func TestPlayerDataRecalculationBatchUsecase_フィンガープリントで列挙し更新に記録する(t *testing.T) {
	// Given
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	snapshot := batchSnapshotForSlotTest(2)
	expected, err := computeMasterFingerprint(snapshot)
	require.NoError(t, err)
	repo := &batchRepositoryStub{
		snapshot: snapshot,
		keys:     []repository.PlayerBatchKey{{ID: 1}},
		data:     map[int]repository.PlayerBatchData{1: {ID: 1}},
	}
	usecase := NewPlayerDataRecalculationBatchUsecase(repo)
	usecase.now = func() time.Time { return time.Date(2026, 7, 6, 12, 0, 0, 0, jst) }

	// When
	result, err := usecase.Execute(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, expected, repo.listFingerprint)
	assert.Equal(t, expected, repo.updates[1].MasterFingerprint)
	assert.Equal(t, expected, result.MasterFingerprint)
}

func TestPlayerDataRecalculationBatchUsecase_運用日が変わってもフィンガープリントは変わらない(t *testing.T) {
	// Given
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	fingerprints := make([]masterfingerprint.Fingerprint, 0, 2)
	for _, now := range []time.Time{time.Date(2026, 7, 6, 12, 0, 0, 0, jst), time.Date(2026, 7, 7, 12, 0, 0, 0, jst)} {
		repo := &batchRepositoryStub{snapshot: batchSnapshotForSlotTest(2)}
		usecase := NewPlayerDataRecalculationBatchUsecase(repo)
		usecase.now = func() time.Time { return now }

		// When
		result, err := usecase.Execute(context.Background())

		require.NoError(t, err)
		fingerprints = append(fingerprints, result.MasterFingerprint)
	}

	// Then
	assert.Equal(t, fingerprints[0], fingerprints[1])
}

func TestPreparedBatchSnapshot_再構築では変わった枠だけを更新する(t *testing.T) {
	bestSlotID := validBatchSnapshot().SlotIDs["best"]
	tests := []struct {
		name            string
		records         []repository.PlayerBatchRecord
		wantClear       []int
		wantAssignments []repository.PlayerBatchSlotAssignment
	}{
		{
			name: "枠が同じなら更新しない",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
				{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(2)},
			},
			wantClear:       []int{},
			wantAssignments: []repository.PlayerBatchSlotAssignment{},
		},
		{
			name: "順位だけ変わった譜面は付け替えだけ行う",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(2)},
				{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
			},
			wantClear: []int{},
			wantAssignments: []repository.PlayerBatchSlotAssignment{
				{ChartID: 1, SlotID: bestSlotID, Position: 1},
				{ChartID: 2, SlotID: bestSlotID, Position: 2},
			},
		},
		{
			name: "枠から外れた譜面を外す",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
				{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(2)},
				{ChartID: 3, Score: 0, SlotName: "best_candidate", SlotOrder: batchIntPtr(1)},
			},
			wantClear:       []int{3},
			wantAssignments: []repository.PlayerBatchSlotAssignment{},
		},
		{
			name: "noneなのに順位がある譜面を外す",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
				{ChartID: 2, Score: 1_008_000, SlotName: "best", SlotOrder: batchIntPtr(2)},
				{ChartID: 3, Score: 0, SlotName: "none", SlotOrder: batchIntPtr(3)},
			},
			wantClear:       []int{3},
			wantAssignments: []repository.PlayerBatchSlotAssignment{},
		},
		{
			name: "枠のない譜面を新しく割り当てる",
			records: []repository.PlayerBatchRecord{
				{ChartID: 1, Score: 1_009_000, SlotName: "best", SlotOrder: batchIntPtr(1)},
				{ChartID: 2, Score: 1_008_000, SlotName: "none"},
			},
			wantClear:       []int{},
			wantAssignments: []repository.PlayerBatchSlotAssignment{{ChartID: 2, SlotID: bestSlotID, Position: 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			// 譜面3は削除済み楽曲にして枠の対象外にします。
			snapshot := batchSnapshotForSlotTest(3)
			snapshot.Songs[2].IsDeleted = true
			prepared := preparedBatchSnapshotForCustomSnapshot(t, snapshot)

			// When
			update, _, err := prepared.buildUpdate(repository.PlayerBatchData{ID: 1, Records: tt.records}, false)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wantClear, update.ClearChartIDs)
			assert.Equal(t, tt.wantAssignments, update.Assignments)
		})
	}
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
	listFingerprint masterfingerprint.Fingerprint
	keys            []repository.PlayerBatchKey
	data            map[int]repository.PlayerBatchData
	updates         map[int]repository.PlayerBatchUpdate
	afterBuildErr   error
}

func (s *batchRepositoryStub) LoadSnapshot(_ context.Context, operationalDate time.Time) (repository.PlayerDataMasterSnapshot, error) {
	s.operationalDate = operationalDate
	return s.snapshot, nil
}

func (s *batchRepositoryStub) ListPlayerKeys(_ context.Context, afterID, _ int, _ int, fingerprint masterfingerprint.Fingerprint) ([]repository.PlayerBatchKey, error) {
	s.listFingerprint = fingerprint
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
