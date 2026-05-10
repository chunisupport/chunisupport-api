package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	mastervo "github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyScores_通常譜面とWORLDSENDを保存し通常譜面だけをOVERPOWER集計する(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 4, 27, 12, 34, 56, 0, time.UTC)
	repo := &stubPlayerDataRepositoryForApplyScoresTest{
		overpowerStats: &repository.OverpowerTargetStats{
			MaxOverpowerTotal: service.CalcSongMaxOP(15.0) + service.CalcSongMaxOP(14.0),
		},
	}
	uc := &playerDataUsecase{playerDataRepo: repo}
	masters := newApplyScoresTestMasters()
	payload := PlayerDataScorePayload{
		Full: []PlayerDataScoreEntry{
			{
				Idx:       "full-song",
				Diff:      "MAS",
				Score:     1010000,
				ClearLamp: stringPtrForApplyScoresTest("absolute"),
				ComboLv:   intPtrForApplyScoresTest(3),
				FullChain: intPtrForApplyScoresTest(3),
				Slot:      stringPtrForApplyScoresTest("best"),
				Order:     intPtrForApplyScoresTest(2),
			},
		},
		Worldsend: []PlayerDataScoreEntry{
			{
				Idx:       "world-song",
				Score:     1009000,
				ClearLamp: stringPtrForApplyScoresTest("hard"),
				ComboLv:   intPtrForApplyScoresTest(2),
				FullChain: intPtrForApplyScoresTest(2),
			},
		},
	}

	// When
	counts, skipped, overpower, err := uc.applyScores(context.Background(), nil, 99, payload, masters, updatedAt)

	// Then
	require.NoError(t, err)
	assert.Equal(t, api_internal.PlayerDataCounts{
		FullRecordsUpserted:      1,
		WorldsendRecordsUpserted: 1,
	}, counts)
	assert.Empty(t, skipped)
	require.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, repository.OverpowerTargetFilter{ExcludeWorldsend: true, ExcludeDeleted: true, PlayerID: intPtrForApplyScoresTest(99)}, repo.receivedFilter)

	require.Len(t, repo.savedInput.FullRecords, 1)
	assert.Equal(t, repository.PlayerRecordForUpsert{
		PlayerID: 99,
		ChartID:  101,
		State: repository.PlayerRecordState{
			Score:       1010000,
			ClearLampID: 5,
			ComboLampID: 3,
			FullChainID: 3,
			SlotID:      2,
			SlotOrder:   intPtrForApplyScoresTest(2),
			UpdatedAt:   updatedAt,
		},
	}, repo.savedInput.FullRecords[0])

	require.Len(t, repo.savedInput.WorldsendRecords, 1)
	assert.Equal(t, repository.WorldsendRecordForUpsert{
		PlayerID: 99,
		ChartID:  201,
		State: repository.WorldsendRecordState{
			Score:       1009000,
			ClearLampID: 3,
			ComboLampID: 2,
			FullChainID: 2,
			UpdatedAt:   updatedAt,
		},
	}, repo.savedInput.WorldsendRecords[0])

	require.NotNil(t, overpower.Value)
	require.NotNil(t, overpower.Percent)
	wantValue := service.CalcSingleOverpower(1010000, 15.0, 3)
	wantPercent := roundFloat(wantValue/repo.overpowerStats.MaxOverpowerTotal*100, 4)
	assert.InDelta(t, wantValue, *overpower.Value, 0.0001)
	assert.InDelta(t, wantPercent, *overpower.Percent, 0.0001)
}

func TestApplyScores_不正レコードをスキップして理由を保持する(t *testing.T) {
	tests := []struct {
		name              string
		payload           PlayerDataScorePayload
		wantCounts        api_internal.PlayerDataCounts
		wantSkipped       []api_internal.SkippedRecord
		wantFullRecords   []repository.PlayerRecordForUpsert
		wantWorldsendRows []repository.WorldsendRecordForUpsert
	}{
		{
			name: "通常譜面で譜面解決失敗と範囲外スコアとスロット解決失敗を返す",
			payload: PlayerDataScorePayload{
				Full: []PlayerDataScoreEntry{
					{Idx: "missing-song", Diff: "MAS", Score: 1000000},
					{Idx: "full-song", Diff: "MAS", Score: 1010001},
					{Idx: "full-song", Diff: "MAS", Score: 1009000, Slot: stringPtrForApplyScoresTest("unknown")},
				},
			},
			wantCounts: api_internal.PlayerDataCounts{
				FullRecordsUpserted: 3,
				FullRecordsSkipped:  3,
			},
			wantSkipped: []api_internal.SkippedRecord{
				{RecordType: "full", Reason: "failed to resolve chart", Details: "idx=missing-song, diff=MAS, error=resource not found: song(missing-song)"},
				{RecordType: "full", Reason: "score out of range: 1010001", Details: "idx=full-song (Full Song), score=1010001"},
				{RecordType: "full", Reason: "failed to resolve slot", Details: "idx=full-song (Full Song), slot=unknown, error=resource not found: slot(unknown)"},
			},
			wantFullRecords:   []repository.PlayerRecordForUpsert{},
			wantWorldsendRows: []repository.WorldsendRecordForUpsert{},
		},
		{
			name: "WORLDSENDで譜面解決失敗とランプ解決失敗を返す",
			payload: PlayerDataScorePayload{
				Worldsend: []PlayerDataScoreEntry{
					{Idx: "missing-world", Score: 1000000},
					{Idx: "world-song", Score: 1000000, ClearLamp: stringPtrForApplyScoresTest("unknown")},
					{Idx: "world-song", Score: 1000000, ComboLv: intPtrForApplyScoresTest(9)},
					{Idx: "world-song", Score: 1000000, FullChain: intPtrForApplyScoresTest(9)},
				},
			},
			wantCounts: api_internal.PlayerDataCounts{
				WorldsendRecordsUpserted: 4,
				WorldsendRecordsSkipped:  4,
			},
			wantSkipped: []api_internal.SkippedRecord{
				{RecordType: "worldsend", Reason: "failed to resolve worldsend chart", Details: "idx=missing-world, error=resource not found: song(missing-world)"},
				{RecordType: "worldsend", Reason: "failed to resolve clear_lamp", Details: "idx=world-song (World Song), clear_lamp=unknown, error=resource not found: clear_lamp(UNKNOWN)"},
				{RecordType: "worldsend", Reason: "failed to resolve combo_lamp", Details: "idx=world-song (World Song), combo_lv=9, error=cmb_lv: unknown combo level: 9"},
				{RecordType: "worldsend", Reason: "failed to resolve full_chain", Details: "idx=world-song (World Song), full_chain=9, error=fch_lv: unknown full chain level: 9"},
			},
			wantFullRecords:   []repository.PlayerRecordForUpsert{},
			wantWorldsendRows: []repository.WorldsendRecordForUpsert{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubPlayerDataRepositoryForApplyScoresTest{
				overpowerStats: &repository.OverpowerTargetStats{MaxOverpowerTotal: service.CalcSongMaxOP(15.0)},
			}
			uc := &playerDataUsecase{playerDataRepo: repo}

			// When
			counts, skipped, overpower, err := uc.applyScores(context.Background(), nil, 77, tt.payload, newApplyScoresTestMasters(), time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC))

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wantCounts, counts)
			assert.Equal(t, tt.wantSkipped, skipped)
			assert.Equal(t, tt.wantFullRecords, repo.savedInput.FullRecords)
			assert.Equal(t, tt.wantWorldsendRows, repo.savedInput.WorldsendRecords)
			require.NotNil(t, overpower.Value)
			require.NotNil(t, overpower.Percent)
			assert.Equal(t, 0.0, *overpower.Value)
			assert.Equal(t, 0.0, *overpower.Percent)
		})
	}
}

type stubPlayerDataRepositoryForApplyScoresTest struct {
	savedInput     repository.PlayerDataSaveInput
	receivedFilter repository.OverpowerTargetFilter
	overpowerStats *repository.OverpowerTargetStats
	saveCalls      int
	saveErr        error
	overpowerErr   error
}

func (s *stubPlayerDataRepositoryForApplyScoresTest) LoadMasterData(_ context.Context, _ []string) (*repository.PlayerDataMaster, error) {
	return nil, nil
}

func (s *stubPlayerDataRepositoryForApplyScoresTest) SavePlayerData(_ context.Context, _ repository.Executor, input repository.PlayerDataSaveInput) error {
	s.saveCalls++
	s.savedInput = input
	return s.saveErr
}

func (s *stubPlayerDataRepositoryForApplyScoresTest) GetOverpowerTargetStats(_ context.Context, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	s.receivedFilter = filter
	if s.overpowerStats == nil {
		return &repository.OverpowerTargetStats{}, s.overpowerErr
	}
	return s.overpowerStats, s.overpowerErr
}

func newApplyScoresTestMasters() *playerDataMaster {
	return &playerDataMaster{
		PlayerDataMasters: &domainmasterdata.PlayerDataMasters{
			ClearLamps: map[string]mastervo.ClearLampType{
				"failed":      {ID: 1, Name: "FAILED"},
				"clear":       {ID: 2, Name: "CLEAR"},
				"hard":        {ID: 3, Name: "HARD"},
				"brave":       {ID: 4, Name: "BRAVE"},
				"absolute":    {ID: 5, Name: "ABSOLUTE"},
				"catastrophy": {ID: 6, Name: "CATASTROPHY"},
			},
			ComboLamps: map[string]mastervo.ComboLampType{
				"none":        {ID: 1, Name: "none"},
				"full combo":  {ID: 2, Name: "full combo"},
				"all justice": {ID: 3, Name: "all justice"},
			},
			FullChains: map[string]mastervo.FullChainType{
				"none":                {ID: 1, Name: "none"},
				"full chain gold":     {ID: 2, Name: "full chain gold"},
				"full chain platinum": {ID: 3, Name: "full chain platinum"},
			},
			Slots: map[string]mastervo.Slot{
				"none": {ID: 1, Name: "none"},
				"best": {ID: 2, Name: "best"},
			},
			Difficulties: map[string]mastervo.ChartDifficulty{
				"MASTER": {ID: 4, Name: "MASTER", SortOrder: 4},
			},
		},
		songs: map[string]entity.PlayerDataSong{
			"full-song":  {ID: 1, OfficialIdx: "full-song", Title: "Full Song"},
			"world-song": {ID: 2, OfficialIdx: "world-song", Title: "World Song"},
		},
		chartsByKey: map[string]entity.PlayerDataChart{
			"1:4": {ID: 101, SongID: 1, DifficultyID: 4, Const: chartconstant.ChartConstant(15.0)},
		},
		chartsByID: map[int]entity.PlayerDataChart{
			101: {ID: 101, SongID: 1, DifficultyID: 4, Const: chartconstant.ChartConstant(15.0)},
		},
		worldsendBySongID: map[int]entity.PlayerDataWorldsendChart{
			2: {ID: 201, SongID: 2},
		},
	}
}

func stringPtrForApplyScoresTest(value string) *string {
	return &value
}

func intPtrForApplyScoresTest(value int) *int {
	return &value
}
