package api_v1

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

func TestToV1WorldsendSongDTO(t *testing.T) {
	genreID := 2
	releasedAt := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	level, err := levelstar.NewLevelStar(3)
	require.NoError(t, err)
	attribute := "光"
	chartNotes, err := notes.NewNotes(2000)
	require.NoError(t, err)
	song := &entity.Song{GenreID: &genreID, ReleasedAt: &releasedAt}
	chart := &entity.WorldsendChart{
		LevelStar: &level,
		Attribute: &attribute,
		Notes:     &chartNotes,
	}

	dto := ToV1WorldsendSongDTO(song, chart, map[int]string{2: "niconico"})

	require.NotNil(t, dto)
	require.NotNil(t, dto.Genre)
	assert.Equal(t, "niconico", *dto.Genre)
	require.NotNil(t, dto.Release)
	assert.Equal(t, "2023-12-31", *dto.Release)
	require.Contains(t, dto.Charts, "WORLDSEND")
	require.NotNil(t, dto.Charts["WORLDSEND"])
	assert.Equal(t, 3, *dto.Charts["WORLDSEND"].LevelStar)
	assert.Equal(t, "光", *dto.Charts["WORLDSEND"].Attribute)
	assert.Equal(t, 2000, *dto.Charts["WORLDSEND"].Notes)
}

// TestToV1WorldsendSongDTO_NilSong は Song が nil の場合に nil を返すことを確認します。
func TestToV1WorldsendSongDTO_NilSong(t *testing.T) {
	dto := ToV1WorldsendSongDTO(nil, nil, map[int]string{})
	if dto != nil {
		assert.Failf(t, "アサーション失敗", "expected nil, got %v", dto)
	}
}

// TestV1WorldsendSongDTO_JSONMarshal は V1WorldsendSongDTO のJSONマーシャリングをテストします。
func TestV1WorldsendSongDTO_JSONMarshal(t *testing.T) {
	releaseDate := "2024-06-01"
	jacket := "we_jacket.png"
	bpm := 160
	genre := "VARIETY"
	reading := "ブイワンワールドエンドテスト"
	levelStar := 4
	attribute := "蔵"
	notesVal := 800
	notesDesigner := "譜面作者B"

	songDTO := &V1WorldsendSongDTO{
		DisplayID:   "v1we123456789012",
		Title:       "V1 WE テスト",
		Reading:     &reading,
		Artist:      "V1 WE アーティスト",
		Genre:       &genre,
		BPM:         &bpm,
		Release:     &releaseDate,
		Jacket:      &jacket,
		OfficialIdx: "789",
		Charts: map[string]*V1WorldsendChartDTO{
			"WORLDSEND": {
				Attribute:     &attribute,
				LevelStar:     &levelStar,
				Notes:         &notesVal,
				NotesDesigner: &notesDesigner,
			},
		},
	}

	jsonBytes, err := json.Marshal(songDTO)
	if err != nil {
		require.Failf(t, "前提条件失敗", "json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	// release フィールド名であることを確認
	if !containsString(jsonString, `"release":"2024-06-01"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'release' field, got: %s", jsonString)
	}

	if !containsString(jsonString, `"reading":"ブイワンワールドエンドテスト"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain reading field, got: %s", jsonString)
	}

	// genre がジャンル名であることを確認
	if !containsString(jsonString, `"genre":"VARIETY"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain genre name, got: %s", jsonString)
	}

	// official_idx が含まれること
	if !containsString(jsonString, `"official_idx":"789"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain official_idx, got: %s", jsonString)
	}

	// charts.WORLDSEND が含まれること
	if !containsString(jsonString, `"WORLDSEND"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'WORLDSEND' key, got: %s", jsonString)
	}

	// attribute, level_star, notes が含まれること
	if !containsString(jsonString, `"attribute":"蔵"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain attribute, got: %s", jsonString)
	}

	if !containsString(jsonString, `"level_star":4`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain level_star, got: %s", jsonString)
	}

	if !containsString(jsonString, `"notes":800`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain notes, got: %s", jsonString)
	}
	if !containsString(jsonString, `"notes_designer":"譜面作者B"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain notes_designer, got: %s", jsonString)
	}
}
