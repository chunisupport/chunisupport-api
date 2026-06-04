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
	uc := &playerDataUsecase{
		playerDataRepo: repo,
		playerRecRepo: &stubPlayerRecordRepositoryForApplyScoresTest{
			records: []*entity.PlayerRecord{{
				Score:       1010000,
				ComboLampID: 3,
				Song:        &entity.Song{ID: 1},
				Chart:       &entity.Chart{Const: chartconstant.ChartConstant(15.0)},
			}},
		},
	}
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
	counts, skipped, changes, overpower, err := uc.applyScores(context.Background(), nil, 99, payload, masters, updatedAt)

	// Then
	require.NoError(t, err)
	assert.Equal(t, api_internal.PlayerDataCounts{
		FullRecordsUpserted:             1,
		WorldsendRecordsUpserted:        1,
		FullRecordsActuallyChanged:      1,
		WorldsendRecordsActuallyChanged: 1,
	}, counts)
	assert.Empty(t, skipped)
	assert.Len(t, changes, 2)
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
			FullChainID: 2,
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
			FullChainID: 3,
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

func TestApplyScores_既存レコードを含めてOVERPOWERを再計算する(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 4, 27, 12, 34, 56, 0, time.UTC)
	repo := &stubPlayerDataRepositoryForApplyScoresTest{
		overpowerStats: &repository.OverpowerTargetStats{
			MaxOverpowerTotal: service.CalcSongMaxOP(15.0) + service.CalcSongMaxOP(14.0),
		},
	}
	playerRecRepo := &stubPlayerRecordRepositoryForApplyScoresTest{
		records: []*entity.PlayerRecord{
			{
				Score:       1010000,
				ComboLampID: 3,
				Song:        &entity.Song{ID: 1},
				Chart:       &entity.Chart{Const: chartconstant.ChartConstant(15.0)},
			},
			{
				Score:       1009000,
				ComboLampID: 3,
				Song:        &entity.Song{ID: 2},
				Chart:       &entity.Chart{Const: chartconstant.ChartConstant(14.0)},
			},
		},
	}
	uc := &playerDataUsecase{playerDataRepo: repo, playerRecRepo: playerRecRepo}
	masters := newApplyScoresTestMasters()
	payload := PlayerDataScorePayload{
		Full: []PlayerDataScoreEntry{{
			Idx: "full-song", Diff: "MAS", Score: 1010000, ComboLv: intPtrForApplyScoresTest(3),
		}},
	}

	// When
	_, _, _, overpower, err := uc.applyScores(context.Background(), nil, 99, payload, masters, updatedAt)

	// Then
	require.NoError(t, err)
	require.NotNil(t, overpower)
	require.NotNil(t, overpower.Value)
	require.NotNil(t, overpower.Percent)
	wantValue := service.CalcSingleOverpower(1010000, 15.0, 3) + service.CalcSingleOverpower(1009000, 14.0, 3)
	wantPercent := roundFloat(wantValue/repo.overpowerStats.MaxOverpowerTotal*100, 4)
	assert.InDelta(t, wantValue, *overpower.Value, 0.0001)
	assert.InDelta(t, wantPercent, *overpower.Percent, 0.0001)
}

func TestApplyScores_未解禁曲を除外してOVERPOWERを再計算する(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 4, 27, 12, 34, 56, 0, time.UTC)
	repo := &stubPlayerDataRepositoryForApplyScoresTest{
		overpowerStats: &repository.OverpowerTargetStats{
			MaxOverpowerTotal: service.CalcSongMaxOP(15.0) + service.CalcSongMaxOP(13.0),
		},
	}
	playerRecRepo := &stubPlayerRecordRepositoryForApplyScoresTest{
		records: []*entity.PlayerRecord{
			{
				Score:           1010000,
				ComboLampID:     3,
				Song:            &entity.Song{ID: 1},
				Chart:           &entity.Chart{Const: chartconstant.ChartConstant(15.0)},
				ChartDifficulty: &entity.ChartDifficulty{Name: "MASTER"},
			},
			{
				Score:           1010000,
				ComboLampID:     3,
				Song:            &entity.Song{ID: 2},
				Chart:           &entity.Chart{Const: chartconstant.ChartConstant(14.0)},
				ChartDifficulty: &entity.ChartDifficulty{Name: "MASTER"},
			},
			{
				Score:           1009000,
				ComboLampID:     3,
				Song:            &entity.Song{ID: 3},
				Chart:           &entity.Chart{Const: chartconstant.ChartConstant(13.0)},
				ChartDifficulty: &entity.ChartDifficulty{Name: "MASTER"},
			},
			{
				Score:           1010000,
				ComboLampID:     3,
				Song:            &entity.Song{ID: 3},
				Chart:           &entity.Chart{Const: chartconstant.ChartConstant(15.0)},
				ChartDifficulty: &entity.ChartDifficulty{Name: "ULTIMA"},
			},
		},
	}
	lockedRepo := &stubPlayerLockedSongRepositoryForApplyScoresTest{
		lockedSongs: []*entity.PlayerLockedSong{
			{PlayerID: 99, SongID: 2, IsUltima: false},
			{PlayerID: 99, SongID: 3, IsUltima: true},
		},
	}
	uc := &playerDataUsecase{playerDataRepo: repo, playerRecRepo: playerRecRepo, lockedRepo: lockedRepo}

	// When
	_, _, _, overpower, err := uc.applyScores(context.Background(), nil, 99, PlayerDataScorePayload{}, newApplyScoresTestMasters(), updatedAt)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 99, lockedRepo.receivedPlayerID)
	require.NotNil(t, overpower.Value)
	require.NotNil(t, overpower.Percent)
	wantValue := service.CalcSingleOverpower(1010000, 15.0, 3) + service.CalcSingleOverpower(1009000, 13.0, 3)
	wantPercent := roundFloat(wantValue/repo.overpowerStats.MaxOverpowerTotal*100, 4)
	assert.InDelta(t, wantValue, *overpower.Value, 0.0001)
	assert.InDelta(t, wantPercent, *overpower.Percent, 0.0001)
}

func TestApplyScores_既存レコード取得失敗時はエラーを返す(t *testing.T) {
	// Given
	repo := &stubPlayerDataRepositoryForApplyScoresTest{
		overpowerStats: &repository.OverpowerTargetStats{MaxOverpowerTotal: service.CalcSongMaxOP(15.0)},
	}
	uc := &playerDataUsecase{
		playerDataRepo: repo,
		playerRecRepo:  &stubPlayerRecordRepositoryForApplyScoresTest{err: context.DeadlineExceeded},
	}

	// When
	_, _, _, _, err := uc.applyScores(context.Background(), nil, 99, PlayerDataScorePayload{}, newApplyScoresTestMasters(), time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC))

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestApplyScores_既存レコードの関連情報欠損時はスキップして集計する(t *testing.T) {
	// Given
	repo := &stubPlayerDataRepositoryForApplyScoresTest{
		overpowerStats: &repository.OverpowerTargetStats{MaxOverpowerTotal: service.CalcSongMaxOP(15.0)},
	}
	uc := &playerDataUsecase{
		playerDataRepo: repo,
		playerRecRepo: &stubPlayerRecordRepositoryForApplyScoresTest{
			records: []*entity.PlayerRecord{{
				Score:       1009000,
				ComboLampID: 2,
				Song:        nil,
				Chart:       &entity.Chart{Const: chartconstant.ChartConstant(15.0)},
			}},
		},
	}

	// When
	_, _, _, overpower, err := uc.applyScores(context.Background(), nil, 99, PlayerDataScorePayload{}, newApplyScoresTestMasters(), time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC))

	// Then
	require.NoError(t, err)
	require.NotNil(t, overpower)
	require.NotNil(t, overpower.Value)
	require.NotNil(t, overpower.Percent)
	assert.InDelta(t, 0.0, *overpower.Value, 0.0001)
	assert.InDelta(t, 0.0, *overpower.Percent, 0.0001)
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
			uc := &playerDataUsecase{playerDataRepo: repo, playerRecRepo: &stubPlayerRecordRepositoryForApplyScoresTest{}}

			// When
			counts, skipped, changes, overpower, err := uc.applyScores(context.Background(), nil, 77, tt.payload, newApplyScoresTestMasters(), time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC))

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wantCounts, counts)
			assert.Equal(t, tt.wantSkipped, skipped)
			assert.Equal(t, tt.wantFullRecords, repo.savedInput.FullRecords)
			assert.Equal(t, tt.wantWorldsendRows, repo.savedInput.WorldsendRecords)
			assert.Empty(t, changes)
			require.NotNil(t, overpower.Value)
			require.NotNil(t, overpower.Percent)
			assert.Equal(t, 0.0, *overpower.Value)
			assert.Equal(t, 0.0, *overpower.Percent)
		})
	}
}

func TestResolveFullChainID_fchLvの値に応じてフルチェインIDを解決する(t *testing.T) {
	tests := []struct {
		name        string
		fullChain   *int
		expectedID  int
		expectedErr string
	}{
		{
			name:       "fch_lvがnilの場合はnoneとして解決される",
			fullChain:  nil,
			expectedID: 1,
		},
		{
			name:       "fch_lvが1の場合はnoneとして解決される",
			fullChain:  intPtrForApplyScoresTest(1),
			expectedID: 1,
		},
		{
			name:       "fch_lvが2の場合はfull chain platinumとして解決される",
			fullChain:  intPtrForApplyScoresTest(2),
			expectedID: 3,
		},
		{
			name:       "fch_lvが3の場合はfull chain goldとして解決される",
			fullChain:  intPtrForApplyScoresTest(3),
			expectedID: 2,
		},
		{
			name:        "未知のfch_lvはバリデーションエラーになる",
			fullChain:   intPtrForApplyScoresTest(9),
			expectedErr: "fch_lv: unknown full chain level: 9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			masters := newApplyScoresTestMasters()

			// When
			id, err := resolveFullChainID(tt.fullChain, masters)

			// Then
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedID, id)
		})
	}
}

type stubPlayerDataRepositoryForApplyScoresTest struct {
	savedInput      repository.PlayerDataSaveInput
	receivedFilter  repository.OverpowerTargetFilter
	overpowerStats  *repository.OverpowerTargetStats
	saveCalls       int
	saveErr         error
	overpowerErr    error
	fullBefore      map[int]repository.PlayerRecordState
	worldsendBefore map[int]repository.WorldsendRecordState
}

func (s *stubPlayerDataRepositoryForApplyScoresTest) FindPlayerRecordStatesByChartIDs(_ context.Context, _ repository.Executor, _ int, _ []int) (map[int]repository.PlayerRecordState, error) {
	if s.fullBefore == nil {
		return map[int]repository.PlayerRecordState{}, nil
	}
	return s.fullBefore, nil
}

func (s *stubPlayerDataRepositoryForApplyScoresTest) FindWorldsendRecordStatesByChartIDs(_ context.Context, _ repository.Executor, _ int, _ []int) (map[int]repository.WorldsendRecordState, error) {
	if s.worldsendBefore == nil {
		return map[int]repository.WorldsendRecordState{}, nil
	}
	return s.worldsendBefore, nil
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

func (s *stubPlayerDataRepositoryForApplyScoresTest) GetOverpowerTargetStatsWithExecutor(ctx context.Context, exec repository.Executor, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	return s.GetOverpowerTargetStats(ctx, filter)
}

type stubPlayerLockedSongRepositoryForApplyScoresTest struct {
	lockedSongs      []*entity.PlayerLockedSong
	receivedPlayerID int
	err              error
}

func (s *stubPlayerLockedSongRepositoryForApplyScoresTest) ListByPlayerID(_ context.Context, _ repository.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	s.receivedPlayerID = playerID
	return s.lockedSongs, s.err
}

func (s *stubPlayerLockedSongRepositoryForApplyScoresTest) Create(_ context.Context, _ repository.Executor, _ *entity.PlayerLockedSong) error {
	return nil
}

func (s *stubPlayerLockedSongRepositoryForApplyScoresTest) Delete(_ context.Context, _ repository.Executor, _ int, _ int, _ bool) error {
	return nil
}

func (s *stubPlayerLockedSongRepositoryForApplyScoresTest) BulkCreate(_ context.Context, _ repository.Executor, _ []*entity.PlayerLockedSong) error {
	return nil
}

func (s *stubPlayerLockedSongRepositoryForApplyScoresTest) BulkDelete(_ context.Context, _ repository.Executor, _ int, _ []int, _ []bool) error {
	return nil
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

type stubPlayerRecordRepositoryForApplyScoresTest struct {
	records []*entity.PlayerRecord
	err     error
}

func (s *stubPlayerRecordRepositoryForApplyScoresTest) FindByPlayerID(_ context.Context, _ repository.Executor, _ int) ([]*entity.PlayerRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records, nil
}

func (s *stubPlayerRecordRepositoryForApplyScoresTest) FindByPlayerIDForRating(_ context.Context, _ repository.Executor, _ int) ([]*entity.PlayerRecord, error) {
	return nil, nil
}

func (s *stubPlayerRecordRepositoryForApplyScoresTest) GetLastScoreUpdate(_ context.Context, _ repository.Executor, _ int) (*time.Time, error) {
	return nil, nil
}

func stringPtrForApplyScoresTest(value string) *string {
	return &value
}

func intPtrForApplyScoresTest(value int) *int {
	return &value
}

func TestApplyScores_保存前状態との差分を返す(t *testing.T) {
	tests := []struct {
		name        string
		fullBefore  map[int]repository.PlayerRecordState
		worldBefore map[int]repository.WorldsendRecordState
		payload     PlayerDataScorePayload
		wantCounts  api_internal.PlayerDataCounts
		wantChanges []api_internal.PlayerDataRecordChange
	}{
		{
			name: "未登録の通常譜面とWORLDSENDはnewになる",
			payload: PlayerDataScorePayload{
				Full:      []PlayerDataScoreEntry{{Idx: "full-song", Diff: "MAS", Score: 1000000}},
				Worldsend: []PlayerDataScoreEntry{{Idx: "world-song", Score: 990000}},
			},
			wantCounts: api_internal.PlayerDataCounts{FullRecordsUpserted: 1, WorldsendRecordsUpserted: 1, FullRecordsActuallyChanged: 1, WorldsendRecordsActuallyChanged: 1},
			wantChanges: []api_internal.PlayerDataRecordChange{
				{RecordType: "full", ChangeType: "new", Idx: "full-song", Diff: "MASTER", After: api_internal.PlayerDataRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1}},
				{RecordType: "worldsend", ChangeType: "new", Idx: "world-song", Diff: "WE", After: api_internal.PlayerDataRecordState{Score: 990000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1}},
			},
		},
		{
			name:       "scoreとランプが同じでslotだけ違う場合は差分なし",
			fullBefore: map[int]repository.PlayerRecordState{101: {Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 1}},
			payload:    PlayerDataScorePayload{Full: []PlayerDataScoreEntry{{Idx: "full-song", Diff: "MAS", Score: 1000000, Slot: stringPtrForApplyScoresTest("best")}}},
			wantCounts: api_internal.PlayerDataCounts{FullRecordsUpserted: 1},
		},
		{
			name:       "同一キー重複は最後の1件だけを保存して差分対象にする",
			fullBefore: map[int]repository.PlayerRecordState{101: {Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1}},
			payload: PlayerDataScorePayload{Full: []PlayerDataScoreEntry{
				{Idx: "full-song", Diff: "MAS", Score: 1000000},
				{Idx: "full-song", Diff: "MAS", Score: 1005000, ComboLv: intPtrForApplyScoresTest(2)},
			}},
			wantCounts: api_internal.PlayerDataCounts{FullRecordsUpserted: 2, FullRecordsActuallyChanged: 1},
			wantChanges: []api_internal.PlayerDataRecordChange{{
				RecordType: "full", ChangeType: "updated", Idx: "full-song", Diff: "MASTER",
				Before: &api_internal.PlayerDataRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1},
				After:  api_internal.PlayerDataRecordState{Score: 1005000, ClearLampID: 1, ComboLampID: 2, FullChainID: 1},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubPlayerDataRepositoryForApplyScoresTest{
				overpowerStats:  &repository.OverpowerTargetStats{},
				fullBefore:      tt.fullBefore,
				worldsendBefore: tt.worldBefore,
			}
			uc := &playerDataUsecase{playerDataRepo: repo, playerRecRepo: &stubPlayerRecordRepositoryForApplyScoresTest{}}

			// When
			counts, skipped, changes, _, err := uc.applyScores(context.Background(), nil, 77, tt.payload, newApplyScoresTestMasters(), time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC))

			// Then
			require.NoError(t, err)
			assert.Empty(t, skipped)
			assert.Equal(t, tt.wantCounts, counts)
			if tt.wantChanges == nil {
				assert.Empty(t, changes)
			} else {
				assert.Equal(t, tt.wantChanges, changes)
			}
			uniqueChartIDs := map[int]struct{}{}
			for _, r := range repo.savedInput.FullRecords {
				uniqueChartIDs[r.ChartID] = struct{}{}
			}
			assert.Len(t, uniqueChartIDs, len(repo.savedInput.FullRecords))
		})
	}
}

func TestPlayerRecordMeaningfullyChanged_DB更新条件と同じ対象だけを比較する(t *testing.T) {
	base := repository.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 1, SlotOrder: intPtrForApplyScoresTest(1), UpdatedAt: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)}
	tests := []struct {
		name  string
		after repository.PlayerRecordState
		want  bool
	}{
		{name: "score差分あり", after: repository.PlayerRecordState{Score: 1000001, ClearLampID: 1, ComboLampID: 1, FullChainID: 1}, want: true},
		{name: "clear lamp差分あり", after: repository.PlayerRecordState{Score: 1000000, ClearLampID: 2, ComboLampID: 1, FullChainID: 1}, want: true},
		{name: "combo lamp差分あり", after: repository.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 2, FullChainID: 1}, want: true},
		{name: "full chain差分あり", after: repository.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 2}, want: true},
		{name: "slot差分のみ", after: repository.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 2}, want: false},
		{name: "updated_at差分のみ", after: repository.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, UpdatedAt: base.UpdatedAt.Add(time.Hour)}, want: false},
		{name: "全値同一", after: base, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, playerRecordMeaningfullyChanged(base, tt.after))
		})
	}
}
