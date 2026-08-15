package chunirec

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	api_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateLevel(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{12.0, 12.0},
		{12.4, 12.0},
		{12.5, 12.5},
		{12.9, 12.5},
		{13.0, 13.0},
		{14.8, 14.5},
	}

	for _, test := range tests {
		result := calculateLevel(test.input)
		assert.Equal(t, test.expected, result, "Input: %f", test.input)
	}
}

func TestToMusicShowResponse(t *testing.T) {
	// テスト用のデータを準備
	genreID := 1
	bpm := 180
	releaseDate := time.Date(2023, 4, 13, 0, 0, 0, 0, time.UTC)
	notesVal, _ := notes.NewNotes(500)

	chartConstBAS, _ := chartconstant.NewChartConstant(8.0)
	chartConstMAS, _ := chartconstant.NewChartConstant(13.7)

	song := &entity.Song{
		ID:          1,
		DisplayID:   "test-song-001",
		Title:       "テスト楽曲",
		Artist:      "テストアーティスト",
		GenreID:     &genreID,
		BPM:         &bpm,
		ReleasedAt:  &releaseDate,
		OfficialIdx: "001",
		Jacket:      nil,
		IsWorldsend: false,
		IsDeleted:   false,
		Charts: []*entity.Chart{
			{
				ID:             1,
				SongID:         1,
				DifficultyID:   1, // BASIC
				Const:          chartConstBAS,
				IsConstUnknown: false,
				Notes:          &notesVal,
			},
			{
				ID:             2,
				SongID:         1,
				DifficultyID:   4, // MASTER
				Const:          chartConstMAS,
				IsConstUnknown: false,
				Notes:          &notesVal,
			},
		},
	}

	masters := &domainmasterdata.SongMasters{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
		},
	}

	// 変換実行
	result := ToMusicShowResponse(song, masters)

	// 検証
	assert.NotNil(t, result)
	assert.Equal(t, "test-song-001", result.Meta.ID)
	assert.Equal(t, "テスト楽曲", result.Meta.Title)
	assert.Equal(t, "テストアーティスト", result.Meta.Artist)
	assert.NotNil(t, result.Meta.Genre)
	assert.Equal(t, "POPS & ANIME", *result.Meta.Genre)
	assert.NotNil(t, result.Meta.BPM)
	assert.Equal(t, float64(180), *result.Meta.BPM)
	assert.NotNil(t, result.Meta.Release)
	assert.Equal(t, "2023-04-13", *result.Meta.Release)

	// 譜面データの検証
	assert.NotNil(t, result.Data.BAS)
	assert.Equal(t, 8.0, result.Data.BAS.Level)
	assert.Equal(t, 8.0, result.Data.BAS.Const)
	assert.False(t, result.Data.BAS.IsConstUnknown)
	assert.NotNil(t, result.Data.BAS.MaxCombo)
	assert.Equal(t, 500, *result.Data.BAS.MaxCombo)

	assert.NotNil(t, result.Data.MAS)
	assert.Equal(t, 13.5, result.Data.MAS.Level) // 13.7 -> 13.5
	assert.Equal(t, 13.7, result.Data.MAS.Const)
	assert.False(t, result.Data.MAS.IsConstUnknown)

	// 存在しない難易度
	assert.Nil(t, result.Data.ADV)
	assert.Nil(t, result.Data.EXP)
	assert.Nil(t, result.Data.ULT)
}

func TestToRecordsShowAllResponse(t *testing.T) {
	// Given
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	updatedAt := time.Date(1970, 1, 1, 9, 0, 0, 0, jst)
	chartConst, err := chartconstant.NewChartConstant(10.7)
	require.NoError(t, err)

	clearLamp := "CLEAR"
	allJustice := "ALL JUSTICE"
	fullChain := "FULL CHAIN GOLD"
	records := []*dto.PlayerRecordDTO{
		{
			UpdatedAt:      &updatedAt,
			IsPlayed:       true,
			Difficulty:     "EXPERT",
			ID:             "6a88218b1a936bd3",
			Title:          "B.B.K.K.B.K.K.",
			Const:          chartConst,
			IsConstUnknown: true,
			Score:          1003215,
			Rating:         11.32,
			ClearLamp:      &clearLamp,
			ComboLamp:      &allJustice,
			FullChain:      &fullChain,
		},
		{
			IsPlayed:   false,
			Difficulty: "MASTER",
			ID:         "未プレイ",
		},
	}

	// When
	result := ToRecordsShowAllResponse(records, map[string]string{
		"6a88218b1a936bd3": "VARIETY",
	}, jst)

	// Then
	require.NotNil(t, result)
	require.Len(t, result.Records, 1)
	record := result.Records[0]
	assert.Equal(t, "6a88218b1a936bd3", record.ID)
	assert.Equal(t, "EXP", record.Diff)
	assert.Equal(t, 10.5, record.Level)
	assert.Equal(t, "B.B.K.K.B.K.K.", record.Title)
	assert.Equal(t, 10.7, record.Const)
	assert.Equal(t, uint32(1003215), record.Score)
	assert.Equal(t, 11.32, record.Rating)
	assert.True(t, record.IsConstUnknown)
	assert.True(t, record.IsClear)
	assert.True(t, record.IsFullCombo)
	assert.True(t, record.IsAllJustice)
	assert.True(t, record.IsFullChain)
	assert.Equal(t, "VARIETY", record.Genre)
	assert.Equal(t, "1970-01-01T09:00:00+0900", record.UpdatedAt)
	assert.True(t, record.IsPlayed)
}

func TestToRecordsShowAllResponse_ComboLampFlags(t *testing.T) {
	tests := []struct {
		name           string
		comboLamp      *string
		wantFullCombo  bool
		wantAllJustice bool
	}{
		{
			name:           "FULL COMBOはFCのみtrue",
			comboLamp:      stringPtr("FULL COMBO"),
			wantFullCombo:  true,
			wantAllJustice: false,
		},
		{
			name:           "ALL JUSTICEはFCとAJがtrue",
			comboLamp:      stringPtr("ALL JUSTICE"),
			wantFullCombo:  true,
			wantAllJustice: true,
		},
		{
			name:           "コンボランプなしはどちらもfalse",
			comboLamp:      nil,
			wantFullCombo:  false,
			wantAllJustice: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			updatedAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
			chartConst, err := chartconstant.NewChartConstant(14.0)
			require.NoError(t, err)
			records := []*dto.PlayerRecordDTO{
				{
					UpdatedAt:  &updatedAt,
					IsPlayed:   true,
					Difficulty: "MAS",
					ID:         "song001",
					Const:      chartConst,
					ComboLamp:  tt.comboLamp,
				},
			}

			// When
			result := ToRecordsShowAllResponse(records, nil, time.UTC)

			// Then
			require.Len(t, result.Records, 1)
			assert.Equal(t, tt.wantFullCombo, result.Records[0].IsFullCombo)
			assert.Equal(t, tt.wantAllJustice, result.Records[0].IsAllJustice)
		})
	}
}

func TestToChunirecUserDTO_プレイヤー未連携ではnullへ変換する(t *testing.T) {
	// Given
	profile := &api_internal.UserProfileWithRecordsDTO{}

	// When
	result := ToChunirecUserDTO(profile, nil, time.UTC)

	// Then
	assert.Nil(t, result)
}

func TestToChunirecUserDTO_内部ユーザーIDを公開しない(t *testing.T) {
	// Given
	profile := &api_internal.UserProfileWithRecordsDTO{
		UserID: 283,
		Player: &dto.PlayerDTO{
			UpdatedAt: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC),
		},
	}

	// When
	result := ToChunirecUserDTO(profile, nil, time.UTC)

	// Then
	require.NotNil(t, result)
	assert.Zero(t, result.UserID)

	body, err := json.Marshal(result)
	require.NoError(t, err)
	var response map[string]any
	require.NoError(t, json.Unmarshal(body, &response))
	userID, exists := response["user_id"]
	require.True(t, exists)
	assert.Equal(t, float64(0), userID)
}

func stringPtr(value string) *string {
	return &value
}
