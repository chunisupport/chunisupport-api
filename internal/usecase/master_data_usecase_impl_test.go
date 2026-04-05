package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// masterDataMasterProviderMock は MasterDataMasterProvider のモックです。
type masterDataMasterProviderMock struct {
	masters *masterdata.MasterDataMasters
}

func (m *masterDataMasterProviderMock) MasterDataMasters() *masterdata.MasterDataMasters {
	return m.masters
}

// chartStatsMasterProviderMock は ChartStatsMasterProvider のモックです。
type chartStatsMasterProviderMock struct {
	bands []*ratingband.RatingBand
}

func (m *chartStatsMasterProviderMock) RatingBands() []*ratingband.RatingBand {
	return m.bands
}

func TestMasterDataUsecase_GetMasterData(t *testing.T) {
	tests := []struct {
		name string
		// Given
		masters     *masterdata.MasterDataMasters
		ratingBands []*ratingband.RatingBand
		// Then
		wantDifficultyIDs  []int
		wantGenreIDs       []int
		wantAccountTypeIDs []int
		wantVersionIDs     []int
		wantAchievementIDs []int
	}{
		{
			name: "難易度はSortOrder昇順に並ぶ",
			masters: &masterdata.MasterDataMasters{
				Difficulties: map[string]master.ChartDifficulty{
					"MASTER":   {ID: 4, Name: "MASTER", SortOrder: 3},
					"BASIC":    {ID: 1, Name: "BASIC", SortOrder: 0},
					"ULTIMA":   {ID: 5, Name: "ULTIMA", SortOrder: 4},
					"EXPERT":   {ID: 3, Name: "EXPERT", SortOrder: 2},
					"ADVANCED": {ID: 2, Name: "ADVANCED", SortOrder: 1},
				},
				Genres:           map[string]master.Genre{},
				AccountTypes:     map[string]master.AccountType{},
				Versions:         map[int]masterdata.Version{},
				AchievementTypes: map[string]masterdata.Item{},
			},
			ratingBands:       nil,
			wantDifficultyIDs: []int{1, 2, 3, 4, 5},
		},
		{
			name: "ジャンルはID昇順に並ぶ",
			masters: &masterdata.MasterDataMasters{
				Genres: map[string]master.Genre{
					"POPS&ANIME": {ID: 2, Name: "POPS&ANIME"},
					"VARIETY":    {ID: 3, Name: "VARIETY"},
					"ORIGINAL":   {ID: 1, Name: "ORIGINAL"},
				},
				Difficulties:     map[string]master.ChartDifficulty{},
				AccountTypes:     map[string]master.AccountType{},
				Versions:         map[int]masterdata.Version{},
				AchievementTypes: map[string]masterdata.Item{},
			},
			ratingBands:  nil,
			wantGenreIDs: []int{1, 2, 3},
		},
		{
			name: "アカウントタイプはID昇順に並ぶ",
			masters: &masterdata.MasterDataMasters{
				AccountTypes: map[string]master.AccountType{
					"ADMIN":  {ID: 3, Name: "ADMIN"},
					"USER":   {ID: 1, Name: "USER"},
					"EDITOR": {ID: 2, Name: "EDITOR"},
				},
				Genres:           map[string]master.Genre{},
				Difficulties:     map[string]master.ChartDifficulty{},
				Versions:         map[int]masterdata.Version{},
				AchievementTypes: map[string]masterdata.Item{},
			},
			ratingBands:        nil,
			wantAccountTypeIDs: []int{1, 2, 3},
		},
		{
			name: "バージョンはID昇順に並ぶ",
			masters: &masterdata.MasterDataMasters{
				Versions: map[int]masterdata.Version{
					3: {ID: 3, Name: "NEW!!!"},
					1: {ID: 1, Name: "CHUNITHM"},
					2: {ID: 2, Name: "PLUS"},
				},
				Genres:           map[string]master.Genre{},
				Difficulties:     map[string]master.ChartDifficulty{},
				AccountTypes:     map[string]master.AccountType{},
				AchievementTypes: map[string]masterdata.Item{},
			},
			ratingBands:    nil,
			wantVersionIDs: []int{1, 2, 3},
		},
		{
			name: "実績タイプはID昇順に並ぶ",
			masters: &masterdata.MasterDataMasters{
				AchievementTypes: map[string]masterdata.Item{
					"SSS+": {ID: 3, Name: "SSS+"},
					"C":    {ID: 1, Name: "C"},
					"S":    {ID: 2, Name: "S"},
				},
				Genres:       map[string]master.Genre{},
				Difficulties: map[string]master.ChartDifficulty{},
				AccountTypes: map[string]master.AccountType{},
				Versions:     map[int]masterdata.Version{},
			},
			ratingBands:        nil,
			wantAchievementIDs: []int{1, 2, 3},
		},
		{
			name:    "mastersがnilの場合は空スライスとRatingBandsのみ返す",
			masters: nil,
			ratingBands: []*ratingband.RatingBand{
				{ID: 1, Label: "RAINBOW", SortOrder: 0},
			},
			wantDifficultyIDs:  nil,
			wantGenreIDs:       nil,
			wantAccountTypeIDs: nil,
			wantVersionIDs:     nil,
			wantAchievementIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := usecase.NewMasterDataUsecase(
				&masterDataMasterProviderMock{masters: tt.masters},
				&chartStatsMasterProviderMock{bands: tt.ratingBands},
			)

			// When
			out := uc.GetMasterData(context.Background())

			// Then
			require.NotNil(t, out)

			if tt.wantDifficultyIDs != nil {
				actualIDs := make([]int, len(out.Difficulties))
				for i, d := range out.Difficulties {
					actualIDs[i] = d.ID
				}
				assert.Equal(t, tt.wantDifficultyIDs, actualIDs, "難易度の順序が一致しない")
			}
			if tt.wantGenreIDs != nil {
				actualIDs := make([]int, len(out.Genres))
				for i, g := range out.Genres {
					actualIDs[i] = g.ID
				}
				assert.Equal(t, tt.wantGenreIDs, actualIDs, "ジャンルの順序が一致しない")
			}
			if tt.wantAccountTypeIDs != nil {
				actualIDs := make([]int, len(out.AccountTypes))
				for i, a := range out.AccountTypes {
					actualIDs[i] = a.ID
				}
				assert.Equal(t, tt.wantAccountTypeIDs, actualIDs, "アカウントタイプの順序が一致しない")
			}
			if tt.wantVersionIDs != nil {
				actualIDs := make([]int, len(out.Versions))
				for i, v := range out.Versions {
					actualIDs[i] = int(v.ID)
				}
				assert.Equal(t, tt.wantVersionIDs, actualIDs, "バージョンの順序が一致しない")
			}
			if tt.wantAchievementIDs != nil {
				actualIDs := make([]int, len(out.AchievementTypes))
				for i, a := range out.AchievementTypes {
					actualIDs[i] = a.ID
				}
				assert.Equal(t, tt.wantAchievementIDs, actualIDs, "実績タイプの順序が一致しない")
			}
		})
	}
}

func TestMasterDataUsecase_GetMasterData_VersionContent(t *testing.T) {
	// Given: バージョンの内容がそのまま返ること
	releasedAt := time.Date(2015, 7, 16, 0, 0, 0, 0, time.UTC)
	uc := usecase.NewMasterDataUsecase(
		&masterDataMasterProviderMock{
			masters: &masterdata.MasterDataMasters{
				Versions: map[int]masterdata.Version{
					1: {ID: 1, Name: "CHUNITHM", ReleasedAt: releasedAt},
				},
				Genres:           map[string]master.Genre{},
				Difficulties:     map[string]master.ChartDifficulty{},
				AccountTypes:     map[string]master.AccountType{},
				AchievementTypes: map[string]masterdata.Item{},
			},
		},
		&chartStatsMasterProviderMock{},
	)

	// When
	out := uc.GetMasterData(context.Background())

	// Then
	require.Len(t, out.Versions, 1)
	assert.Equal(t, uint8(1), out.Versions[0].ID)
	assert.Equal(t, "CHUNITHM", out.Versions[0].Name)
	assert.Equal(t, releasedAt, out.Versions[0].ReleasedAt)
}
