package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	mastervo "github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePlayerDataPayload_AppVersion は、app_verに関係なく登録できることをテストします。
func TestValidatePlayerDataPayload_AppVersionを検証しない(t *testing.T) {
	tests := []struct {
		name       string
		appVersion string
	}{
		{
			name:       "空文字列でも正常",
			appVersion: "",
		},
		{
			name:       "任意の文字列でも正常",
			appVersion: "invalid_version_string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 最小限のペイロードを作成（スコアは空）
			rating := 0.0
			payload := &PlayerDataPayload{
				AppVersion: tt.appVersion,
				Name:       "テストプレイヤー",
				Level:      1,
				Rating:     &rating,
				LastPlayed: "2024/01/01 00:00",
				Overpower: PlayerDataOverpowerPayload{
					Value:      0.0,
					Percentage: 0.0,
				},
				ClassEmblem: PlayerDataClassPayload{
					MedalClass: "none",
					BaseClass:  "none",
				},
				Team: PlayerDataTeamPayload{
					Name:  "none",
					Color: "",
				},
				Honors: map[string]PlayerDataHonorPayload{},
				Scores: PlayerDataScorePayload{
					Standard:  []PlayerDataScoreEntry{},
					Worldsend: []PlayerDataScoreEntry{},
				},
				UpdatedAt: "2024-01-01T00:00:00Z",
			}

			err := validatePlayerDataPayload(payload)
			assert.NoError(t, err)
		})
	}
}

// TestValidatePlayerDataPayload_NilPayload は、payloadがnilの場合のテストです
func TestValidatePlayerDataPayload_NilPayload(t *testing.T) {
	err := validatePlayerDataPayload(nil)
	require.Error(t, err, "validatePlayerDataPayload(nil) should return error")

	var validationErr *PlayerDataValidationError
	require.ErrorAs(t, err, &validationErr, "validatePlayerDataPayload(nil) should return PlayerDataValidationError")
}

func TestValidateScoreEntry_FullChainはAJまたはFCが必要(t *testing.T) {
	tests := []struct {
		name      string
		entry     PlayerDataScoreEntry
		wantError bool
	}{
		{
			name: "FULL CHAIN GOLDでFCの場合は正常",
			entry: PlayerDataScoreEntry{
				Idx:       "full-song",
				Score:     1009000,
				ComboLv:   intPtrForApplyScoresTest(2),
				FullChain: intPtrForApplyScoresTest(3),
			},
			wantError: false,
		},
		{
			name: "FULL CHAIN PLATINUMでAJの場合は正常",
			entry: PlayerDataScoreEntry{
				Idx:       "full-song",
				Score:     1009000,
				ComboLv:   intPtrForApplyScoresTest(3),
				FullChain: intPtrForApplyScoresTest(2),
			},
			wantError: false,
		},
		{
			name: "FULL CHAINでコンボランプなしの場合は矛盾",
			entry: PlayerDataScoreEntry{
				Idx:       "full-song",
				Score:     1009000,
				FullChain: intPtrForApplyScoresTest(3),
			},
			wantError: true,
		},
		{
			name: "FULL CHAINでNONEの場合は矛盾",
			entry: PlayerDataScoreEntry{
				Idx:       "full-song",
				Score:     1009000,
				ComboLv:   intPtrForApplyScoresTest(1),
				FullChain: intPtrForApplyScoresTest(2),
			},
			wantError: true,
		},
		{
			name: "FULL CHAINなしでコンボランプなしの場合は正常",
			entry: PlayerDataScoreEntry{
				Idx:       "full-song",
				Score:     1009000,
				FullChain: intPtrForApplyScoresTest(1),
			},
			wantError: false,
		},
		{
			name: "未知のFULL CHAIN値はランプ解決側で扱うため正常",
			entry: PlayerDataScoreEntry{
				Idx:       "full-song",
				Score:     1009000,
				FullChain: intPtrForApplyScoresTest(9),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScoreEntry(&tt.entry, "standard", 0)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolveClassEmblemIDs(t *testing.T) {
	tests := []struct {
		name        string
		payload     PlayerDataClassPayload
		wantClassID *int
		wantBaseID  *int
	}{
		{
			name: "0埋め2桁の06をinfとして解決できる",
			payload: PlayerDataClassPayload{
				MedalClass: "06",
				BaseClass:  "06",
			},
			wantClassID: intPtrForApplyScoresTest(6),
			wantBaseID:  intPtrForApplyScoresTest(6),
		},
		{
			name: "infの直接指定も従来通り解決できる",
			payload: PlayerDataClassPayload{
				MedalClass: "INF",
				BaseClass:  "inf",
			},
			wantClassID: intPtrForApplyScoresTest(6),
			wantBaseID:  intPtrForApplyScoresTest(6),
		},
		{
			name: "未定義値はnil扱いになる",
			payload: PlayerDataClassPayload{
				MedalClass: "99",
				BaseClass:  "none",
			},
			wantClassID: nil,
			wantBaseID:  nil,
		},
	}

	masters := &playerDataMaster{
		PlayerDataMasters: &domainmasterdata.PlayerDataMasters{
			ClassEmblems: map[string]mastervo.ClassEmblem{
				"1":   {ID: 1, Name: "1"},
				"inf": {ID: 6, Name: "inf"},
			},
			ClassEmblemBases: map[string]mastervo.ClassEmblemBase{
				"1":   {ID: 1, Name: "1"},
				"inf": {ID: 6, Name: "inf"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClassID, gotBaseID, err := resolveClassEmblemIDs(tt.payload, masters)

			require.NoError(t, err)
			assert.Equal(t, tt.wantClassID, gotClassID)
			assert.Equal(t, tt.wantBaseID, gotBaseID)
		})
	}
}

func TestApplyHonors_SP称号はタイトル空文字と画像URLで登録する(t *testing.T) {
	img1 := "https://example.com/sp-1.png"
	img2 := "https://example.com/sp-2.png"
	honorRepo := &stubHonorRepositoryForApplyHonorsTest{}
	uc := &playerDataUsecase{honorRepo: honorRepo}
	masters := newApplyHonorsTestMasters()

	skipped, err := uc.applyHonors(context.Background(), nil, 100, map[string]PlayerDataHonorPayload{
		"1": {Title: "ペイロード上の称号名", Class: "sp", Img: &img1},
		"2": {Class: "sp", Img: &img2},
	}, masters)

	require.NoError(t, err)
	assert.Empty(t, skipped)
	assert.Equal(t, 1, honorRepo.deleteCount)
	require.Len(t, honorRepo.ensureCalls, 2)
	assert.ElementsMatch(t, []string{img1, img2}, honorRepo.ensureImageURLs())
	assert.ElementsMatch(t, []string{"", ""}, honorRepo.ensureTitles())
	assert.Len(t, honorRepo.assignments, 2)
}

func TestApplyHonors_SP称号で画像URLがない場合はスキップする(t *testing.T) {
	honorRepo := &stubHonorRepositoryForApplyHonorsTest{}
	uc := &playerDataUsecase{honorRepo: honorRepo}
	masters := newApplyHonorsTestMasters()

	skipped, err := uc.applyHonors(context.Background(), nil, 100, map[string]PlayerDataHonorPayload{
		"1": {Class: "sp"},
	}, masters)

	require.NoError(t, err)
	require.Len(t, skipped, 1)
	assert.Equal(t, "sp honor image_url is required", skipped[0].Reason)
	assert.Empty(t, honorRepo.ensureCalls)
	assert.Empty(t, honorRepo.assignments)
}

func TestApplyHonors_通常称号はタイトルと空画像URLで登録する(t *testing.T) {
	honorRepo := &stubHonorRepositoryForApplyHonorsTest{}
	uc := &playerDataUsecase{honorRepo: honorRepo}
	masters := newApplyHonorsTestMasters()

	skipped, err := uc.applyHonors(context.Background(), nil, 100, map[string]PlayerDataHonorPayload{
		"1": {Title: "通常称号", Class: "normal"},
	}, masters)

	require.NoError(t, err)
	assert.Empty(t, skipped)
	require.Len(t, honorRepo.ensureCalls, 1)
	assert.Equal(t, "通常称号", honorRepo.ensureCalls[0].title)
	assert.Nil(t, honorRepo.ensureCalls[0].imageURL)
}

func TestApplyHonors_お気に入りからランダムのスロットは更新しない(t *testing.T) {
	honorRepo := &stubHonorRepositoryForApplyHonorsTest{}
	uc := &playerDataUsecase{honorRepo: honorRepo}
	masters := newApplyHonorsTestMasters()

	skipped, err := uc.applyHonors(context.Background(), nil, 100, map[string]PlayerDataHonorPayload{
		"1": {Title: "お気に入りからランダム", Class: "normal"},
		"2": {Title: "更新する称号", Class: "normal"},
	}, masters)

	require.NoError(t, err)
	assert.Empty(t, skipped)
	assert.Equal(t, []int{1}, honorRepo.preservedSlots)
	require.Len(t, honorRepo.ensureCalls, 1)
	assert.Equal(t, "更新する称号", honorRepo.ensureCalls[0].title)
	require.Len(t, honorRepo.assignments, 1)
	assert.Equal(t, 2, honorRepo.assignments[0].Slot)
}

func TestNewPlayerDataUsecase_PlayerRecRepoがnilの場合はpanicする(t *testing.T) {
	assert.PanicsWithValue(t, "player record repository is required", func() {
		NewPlayerDataUsecase(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})
}

type honorEnsureCallForApplyHonorsTest struct {
	title       string
	honorTypeID int
	imageURL    *string
}

type stubHonorRepositoryForApplyHonorsTest struct {
	deleteCount    int
	preservedSlots []int
	ensureCalls    []honorEnsureCallForApplyHonorsTest
	assignments    []repository.HonorAssignment
	nextHonorID    int
}

func (s *stubHonorRepositoryForApplyHonorsTest) FindAll(_ context.Context, _ repository.Executor) ([]*entity.Honor, error) {
	return []*entity.Honor{}, nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) FindByID(_ context.Context, _ repository.Executor, _ int) (*entity.Honor, error) {
	return nil, repository.ErrHonorNotFound
}

func (s *stubHonorRepositoryForApplyHonorsTest) Create(_ context.Context, _ repository.Executor, honor *entity.Honor) (*entity.Honor, error) {
	return honor, nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) Save(_ context.Context, _ repository.Executor, _ *entity.Honor) error {
	return nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) Delete(_ context.Context, _ repository.Executor, _ int) error {
	return nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) EnsureHonor(_ context.Context, _ repository.Executor, title string, honorTypeID int, imageURL *string) (int, error) {
	s.ensureCalls = append(s.ensureCalls, honorEnsureCallForApplyHonorsTest{
		title:       title,
		honorTypeID: honorTypeID,
		imageURL:    imageURL,
	})
	s.nextHonorID++
	return s.nextHonorID, nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) DeletePlayerHonors(_ context.Context, _ repository.Executor, _ int) error {
	s.deleteCount++
	return nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) DeletePlayerHonorsExceptSlots(_ context.Context, _ repository.Executor, _ int, preservedSlots []int) error {
	s.deleteCount++
	s.preservedSlots = append(s.preservedSlots, preservedSlots...)
	return nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) BulkAssignHonors(_ context.Context, _ repository.Executor, assignments []repository.HonorAssignment) error {
	s.assignments = append(s.assignments, assignments...)
	return nil
}

func (s *stubHonorRepositoryForApplyHonorsTest) ensureImageURLs() []string {
	imageURLs := make([]string, 0, len(s.ensureCalls))
	for _, call := range s.ensureCalls {
		if call.imageURL == nil {
			imageURLs = append(imageURLs, "")
			continue
		}
		imageURLs = append(imageURLs, *call.imageURL)
	}
	return imageURLs
}

func (s *stubHonorRepositoryForApplyHonorsTest) ensureTitles() []string {
	titles := make([]string, 0, len(s.ensureCalls))
	for _, call := range s.ensureCalls {
		titles = append(titles, call.title)
	}
	return titles
}

func newApplyHonorsTestMasters() *playerDataMaster {
	return &playerDataMaster{
		PlayerDataMasters: &domainmasterdata.PlayerDataMasters{
			HonorTypes: map[string]mastervo.HonorType{
				"normal": {ID: 1, Name: "normal"},
				"sp":     {ID: 11, Name: "sp"},
			},
		},
	}
}

func TestCalculateOverpowerSummaryFromPlayerRecords(t *testing.T) {
	t.Run("ChartDifficultyがnilのレコードをスキップしWARNログを出力する", func(t *testing.T) {
		// Arrange
		logBuffer := captureDefaultSlog(t)
		recordScore, err := score.NewScore(1_000_000)
		require.NoError(t, err)
		chartConst, err := chartconstant.NewChartConstant(10.0)
		require.NoError(t, err)

		records := []*entity.PlayerRecord{
			// ChartDifficultyがnilのレコード（スキップされる）
			{
				PlayerID: 1,
				ChartID:  10,
				Score:    recordScore,
				Song:     &entity.Song{ID: 100, Title: "テスト曲1"},
				Chart:    &entity.Chart{ID: 10, Const: chartConst},
				// ChartDifficulty: nil
			},
			// 正常なレコード（集計される）
			{
				PlayerID: 1,
				ChartID:  11,
				Score:    recordScore,
				Song:     &entity.Song{ID: 101, Title: "テスト曲2"},
				Chart:    &entity.Chart{ID: 11, Const: chartConst},
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   1,
					Name: "MASTER",
				},
			},
		}

		lockedSongs := []*entity.PlayerLockedSong{
			{PlayerID: 1, SongID: 200, IsUltima: false},
		}

		// Act
		result, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, 100.0)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result.Value)
		assert.NotNil(t, result.Percent)

		// WARNログが出力されていることを確認
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "level=WARN")
		assert.Contains(t, logOutput, "skipped player records during overpower recalculation")
		assert.Contains(t, logOutput, "chart_difficulty_nil")
	})

	t.Run("ロック楽曲のレコードをスキップしWARNログを出力する", func(t *testing.T) {
		// Arrange
		logBuffer := captureDefaultSlog(t)
		recordScore, err := score.NewScore(1_000_000)
		require.NoError(t, err)
		chartConst, err := chartconstant.NewChartConstant(10.0)
		require.NoError(t, err)

		records := []*entity.PlayerRecord{
			// ロックされている楽曲（スキップされる）
			{
				PlayerID: 1,
				ChartID:  10,
				Score:    recordScore,
				Song:     &entity.Song{ID: 100, Title: "ロック曲"},
				Chart:    &entity.Chart{ID: 10, Const: chartConst},
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   1,
					Name: "MASTER",
				},
			},
			// 正常なレコード（集計される）
			{
				PlayerID: 1,
				ChartID:  11,
				Score:    recordScore,
				Song:     &entity.Song{ID: 101, Title: "通常曲"},
				Chart:    &entity.Chart{ID: 11, Const: chartConst},
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   1,
					Name: "MASTER",
				},
			},
		}

		lockedSongs := []*entity.PlayerLockedSong{
			{PlayerID: 1, SongID: 100, IsUltima: false},
		}

		// Act
		result, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, 100.0)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result.Value)
		assert.NotNil(t, result.Percent)

		// WARNログが出力されていることを確認
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "level=WARN")
		assert.Contains(t, logOutput, "skipped player records during overpower recalculation")
		assert.Contains(t, logOutput, "locked_song")
	})

	t.Run("ULTIMAのロック楽曲も正しくスキップする", func(t *testing.T) {
		// Arrange
		logBuffer := captureDefaultSlog(t)
		recordScore, err := score.NewScore(1_000_000)
		require.NoError(t, err)
		chartConst, err := chartconstant.NewChartConstant(10.0)
		require.NoError(t, err)

		records := []*entity.PlayerRecord{
			// ULTIMAでロックされている楽曲（スキップされる）
			{
				PlayerID: 1,
				ChartID:  10,
				Score:    recordScore,
				Song:     &entity.Song{ID: 100, Title: "ULTIMA曲"},
				Chart:    &entity.Chart{ID: 10, Const: chartConst},
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   5,
					Name: "ULTIMA",
				},
			},
			// 同じ曲の別の難易度（スキップされない）
			{
				PlayerID: 1,
				ChartID:  11,
				Score:    recordScore,
				Song:     &entity.Song{ID: 100, Title: "ULTIMA曲"},
				Chart:    &entity.Chart{ID: 11, Const: chartConst},
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   1,
					Name: "MASTER",
				},
			},
		}

		lockedSongs := []*entity.PlayerLockedSong{
			{PlayerID: 1, SongID: 100, IsUltima: true},
		}

		// Act
		result, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, 100.0)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result.Value)
		assert.NotNil(t, result.Percent)

		// WARNログが出力されていることを確認
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "level=WARN")
		assert.Contains(t, logOutput, "skipped player records during overpower recalculation")
		assert.Contains(t, logOutput, "locked_song")
	})

	t.Run("ロック楽曲がない場合はスキップ処理をバイパスする", func(t *testing.T) {
		// Arrange
		logBuffer := captureDefaultSlog(t)
		recordScore, err := score.NewScore(1_000_000)
		require.NoError(t, err)
		chartConst, err := chartconstant.NewChartConstant(10.0)
		require.NoError(t, err)

		records := []*entity.PlayerRecord{
			{
				PlayerID: 1,
				ChartID:  10,
				Score:    recordScore,
				Song:     &entity.Song{ID: 100, Title: "通常曲"},
				Chart:    &entity.Chart{ID: 10, Const: chartConst},
				// ChartDifficultyがnilでもロック楽曲が空ならスキップチェックはバイパスされる
			},
		}

		lockedSongs := []*entity.PlayerLockedSong{}

		// Act
		result, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, 100.0)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result.Value)
		assert.NotNil(t, result.Percent)

		// ロック楽曲がないのでWARNログは出力されない（または別の理由でスキップされた場合のみ出力される）
		logOutput := logBuffer.String()
		// この場合はChartDifficultyがnilでもスキップロジックがバイパスされるため、WARNログは出ない
		assert.NotContains(t, logOutput, "chart_difficulty_nil")
		assert.NotContains(t, logOutput, "locked_song")
	})

	t.Run("スキップレコードが空の場合はWARNログを出力しない", func(t *testing.T) {
		// Arrange
		logBuffer := captureDefaultSlog(t)
		recordScore, err := score.NewScore(1_000_000)
		require.NoError(t, err)
		chartConst, err := chartconstant.NewChartConstant(10.0)
		require.NoError(t, err)

		records := []*entity.PlayerRecord{
			{
				PlayerID: 1,
				ChartID:  10,
				Score:    recordScore,
				Song:     &entity.Song{ID: 100, Title: "通常曲"},
				Chart:    &entity.Chart{ID: 10, Const: chartConst},
				ChartDifficulty: &entity.ChartDifficulty{
					ID:   1,
					Name: "MASTER",
				},
			},
		}

		lockedSongs := []*entity.PlayerLockedSong{
			{PlayerID: 1, SongID: 200, IsUltima: false}, // 別の曲をロック
		}

		// Act
		result, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, 100.0)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result.Value)
		assert.NotNil(t, result.Percent)

		// スキップレコードがないのでWARNログは出力されない
		logOutput := logBuffer.String()
		assert.NotContains(t, logOutput, "skipped player records during overpower recalculation")
	})
}
