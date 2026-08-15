package reiwa

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/stretchr/testify/assert"
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
		{13.7, 13.5},
	}

	for _, test := range tests {
		result := calculateLevel(test.input)
		assert.Equal(t, test.expected, result, "Input: %f", test.input)
	}
}

func TestDifficultyIDToDiff(t *testing.T) {
	tests := []struct {
		id       int
		expected string
	}{
		{1, "BAS"},
		{2, "ADV"},
		{3, "EXP"},
		{4, "MAS"},
		{5, "ULT"},
		{99, ""},
	}

	for _, test := range tests {
		result := difficultyIDToDiff(test.id)
		assert.Equal(t, test.expected, result, "ID: %d", test.id)
	}
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 0, boolToInt(false))
	assert.Equal(t, 1, boolToInt(true))
}

func TestToChunithmRecordOriginalResponse_Basic(t *testing.T) {
	genreID := 2
	bpm := 200
	jacket := "testjacket123"
	releasedAt := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
	notesVal, _ := notes.NewNotes(500)

	constBAS, _ := chartconstant.NewChartConstant(7.0)
	constADV, _ := chartconstant.NewChartConstant(10.5)

	song := &entity.Song{
		DisplayID:   "song001",
		Title:       "テスト曲",
		Artist:      "テストアーティスト",
		GenreID:     &genreID,
		BPM:         &bpm,
		ReleasedAt:  &releasedAt,
		OfficialIdx: "5",
		Jacket:      &jacket,
		IsWorldsend: false,
		Charts: []*entity.Chart{
			{
				DifficultyID:   1,
				Const:          constBAS,
				IsConstUnknown: false,
				Notes:          &notesVal,
			},
			{
				DifficultyID:   4,
				Const:          constADV,
				IsConstUnknown: true,
				Notes:          &notesVal,
			},
		},
	}

	cache := &masterdata.Cache{
		GenreNamesByID: map[int]string{2: "niconico"},
		VersionsByID: map[int]masterdata.Version{
			1: {ID: 1, Name: "CHUNITHM PARADISE", ReleasedAt: time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	result := ToChunithmRecordOriginalResponse([]*entity.Song{song}, cache)

	assert.Len(t, result, 2)

	r0 := result[0]
	assert.Equal(t, "テスト曲", r0.Title)
	assert.Equal(t, "テストアーティスト", r0.Artist)
	assert.Equal(t, "testjacket123", r0.Img)
	assert.Equal(t, "niconico", r0.Genre)
	assert.Equal(t, 7.0, r0.Const)
	assert.Equal(t, 7.0, r0.Level)
	assert.Equal(t, "BAS", r0.Diff)
	assert.Equal(t, 500, r0.Notes)
	assert.Equal(t, 0, r0.Unknown)
	assert.Equal(t, "song001", r0.ChunirecID)
	assert.Equal(t, "5", r0.Idx)
	assert.Equal(t, 200, r0.BPM)
	assert.Equal(t, "PARADISE", r0.Version)
	// 2020-01-15 00:00:00 JST = 2019-01-14 15:00:00 UTC = 1579014000 / 100 = 15790140
	assert.Equal(t, int64(15790140), r0.Release)

	r1 := result[1]
	assert.Equal(t, "MAS", r1.Diff)
	assert.Equal(t, 10.5, r1.Const)
	assert.Equal(t, 10.5, r1.Level)
	assert.Equal(t, 1, r1.Unknown)
}

func TestToChunithmRecordOriginalResponse_GenrePOPSANIME(t *testing.T) {
	genreID := 1
	releasedAt := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	notesVal, _ := notes.NewNotes(100)
	constVal, _ := chartconstant.NewChartConstant(5.0)

	song := &entity.Song{
		DisplayID:   "song002",
		Title:       "POP曲",
		Artist:      "POPアーティスト",
		GenreID:     &genreID,
		ReleasedAt:  &releasedAt,
		OfficialIdx: "10",
		IsWorldsend: false,
		Charts: []*entity.Chart{
			{
				DifficultyID: 1,
				Const:        constVal,
				Notes:        &notesVal,
			},
		},
	}

	cache := &masterdata.Cache{
		GenreNamesByID: map[int]string{1: "POPS&ANIME"},
		VersionsByID:   map[int]masterdata.Version{},
	}

	result := ToChunithmRecordOriginalResponse([]*entity.Song{song}, cache)
	assert.Len(t, result, 1)
	assert.Equal(t, "POPS & ANIME", result[0].Genre)
}

func TestToChunithmRecordOriginalResponse_ExcludesWorldsend(t *testing.T) {
	notesVal, _ := notes.NewNotes(100)
	constVal, _ := chartconstant.NewChartConstant(5.0)

	normalSong := &entity.Song{
		DisplayID:   "normal",
		Title:       "通常曲",
		Artist:      "A",
		ReleasedAt:  &time.Time{},
		OfficialIdx: "1",
		IsWorldsend: false,
		Charts: []*entity.Chart{
			{DifficultyID: 1, Const: constVal, Notes: &notesVal},
		},
	}
	weSong := &entity.Song{
		DisplayID:   "we001",
		Title:       "WE曲",
		Artist:      "B",
		ReleasedAt:  &time.Time{},
		OfficialIdx: "2",
		IsWorldsend: true,
		Charts: []*entity.Chart{
			{DifficultyID: 1, Const: constVal, Notes: &notesVal},
		},
	}

	result := ToChunithmRecordOriginalResponse([]*entity.Song{normalSong, weSong}, nil)
	assert.Len(t, result, 1)
	assert.Equal(t, "通常曲", result[0].Title)
}

func TestToChunithmRecordOriginalResponse_SortOrder(t *testing.T) {
	notesVal, _ := notes.NewNotes(100)
	constVal, _ := chartconstant.NewChartConstant(5.0)

	songs := []*entity.Song{
		{
			DisplayID: "z", Title: "Z曲", OfficialIdx: "10",
			Artist: "A", ReleasedAt: &time.Time{}, IsWorldsend: false,
			Charts: []*entity.Chart{
				{DifficultyID: 4, Const: constVal, Notes: &notesVal},
				{DifficultyID: 1, Const: constVal, Notes: &notesVal},
			},
		},
		{
			DisplayID: "a", Title: "A曲", OfficialIdx: "2",
			Artist: "B", ReleasedAt: &time.Time{}, IsWorldsend: false,
			Charts: []*entity.Chart{
				{DifficultyID: 1, Const: constVal, Notes: &notesVal},
				{DifficultyID: 5, Const: constVal, Notes: &notesVal},
			},
		},
	}

	result := ToChunithmRecordOriginalResponse(songs, nil)
	assert.Len(t, result, 4)

	// idx: "2" (numerically 2) < "10" (numerically 10)
	assert.Equal(t, "2", result[0].Idx)
	assert.Equal(t, "BAS", result[0].Diff)
	assert.Equal(t, "2", result[1].Idx)
	assert.Equal(t, "ULT", result[1].Diff)
	assert.Equal(t, "10", result[2].Idx)
	assert.Equal(t, "BAS", result[2].Diff)
	assert.Equal(t, "10", result[3].Idx)
	assert.Equal(t, "MAS", result[3].Diff)
}

func TestResolveVersionName_TrimCHUNITHMPrefix(t *testing.T) {
	cache := &masterdata.Cache{
		VersionsByID: map[int]masterdata.Version{
			1: {ID: 1, Name: "CHUNITHM NEW", ReleasedAt: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)},
			2: {ID: 2, Name: "CHUNITHM NEW PLUS", ReleasedAt: time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)},
			3: {ID: 3, Name: "CHUNITHM PARADISE LOST", ReleasedAt: time.Date(2020, 10, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	// 2019-06-15 → CHUNITHM NEW PLUS (released 2019-06-01) → "NEW PLUS"
	songTime := time.Date(2019, 6, 15, 0, 0, 0, 0, time.UTC)
	result := resolveVersionName(&songTime, cache)
	assert.Equal(t, "NEW PLUS", result)

	// 2019-03-01 → CHUNITHM NEW (released 2019-01-01) → "NEW"
	songTime2 := time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC)
	result2 := resolveVersionName(&songTime2, cache)
	assert.Equal(t, "NEW", result2)

	// 2020-12-01 → CHUNITHM PARADISE LOST → "PARADISE LOST"
	songTime3 := time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)
	result3 := resolveVersionName(&songTime3, cache)
	assert.Equal(t, "PARADISE LOST", result3)
}

func TestToChunithmRecordOriginalResponse_EmptyList(t *testing.T) {
	result := ToChunithmRecordOriginalResponse(nil, nil)
	assert.Len(t, result, 0)

	result2 := ToChunithmRecordOriginalResponse([]*entity.Song{}, nil)
	assert.Len(t, result2, 0)
}

func TestToChunithmRecordOriginalResponse_SongWithNoCharts(t *testing.T) {
	song := &entity.Song{
		DisplayID: "noidx", Title: "譜面なし", Artist: "A",
		ReleasedAt: &time.Time{}, OfficialIdx: "1", IsWorldsend: false,
		Charts: []*entity.Chart{},
	}

	result := ToChunithmRecordOriginalResponse([]*entity.Song{song}, nil)
	assert.Len(t, result, 0)
}

func TestResolveVersionName_EmptyForOriginal(t *testing.T) {
	cache := &masterdata.Cache{
		VersionsByID: map[int]masterdata.Version{
			1: {ID: 1, Name: "CHUNITHM", ReleasedAt: time.Date(2015, 7, 16, 0, 0, 0, 0, time.UTC)},
		},
	}

	// "CHUNITHM" のプリフィックスを取り除くと空文字
	songTime := time.Date(2015, 8, 1, 0, 0, 0, 0, time.UTC)
	result := resolveVersionName(&songTime, cache)
	assert.Equal(t, "", result)
}

func TestResolveRelease(t *testing.T) {
	// 2020-01-15 00:00:00 UTC → 2020-01-15 00:00:00 JST
	releasedAt := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
	result := resolveRelease(&releasedAt)

	// 2020-01-15 00:00:00 JST = 2020-01-14 15:00:00 UTC
	// Unix: 1579014000
	// / 100: 15790140
	assert.Equal(t, int64(15790140), result)
}

func TestResolveRelease_Nil(t *testing.T) {
	assert.Equal(t, int64(0), resolveRelease(nil))
}

func TestResolveJacket_Nil(t *testing.T) {
	assert.Equal(t, "", resolveJacket(nil))
}

func TestResolveBPM_Nil(t *testing.T) {
	assert.Equal(t, 0, resolveBPM(nil))
}

func TestResolveNotes_Nil(t *testing.T) {
	assert.Equal(t, 0, resolveNotes(nil))
}

func TestResolveVersionName_NilReleasedAt(t *testing.T) {
	assert.Equal(t, "", resolveVersionName(nil, nil))
}

func TestResolveGenreName_NilGenreID(t *testing.T) {
	assert.Equal(t, "", resolveGenreName(nil, nil))
}

func TestResolveGenreName_UnknownID(t *testing.T) {
	genreID := 999
	cache := &masterdata.Cache{
		GenreNamesByID: map[int]string{1: "POPS&ANIME"},
	}
	assert.Equal(t, "", resolveGenreName(&genreID, cache))
}

func TestSortChunithmRecords(t *testing.T) {
	records := ChunithmRecordList{
		{Idx: "10", Diff: "BAS"},
		{Idx: "2", Diff: "MAS"},
		{Idx: "1", Diff: "EXP"},
		{Idx: "2", Diff: "BAS"},
	}

	sortChunithmRecords(records)

	assert.Equal(t, "1", records[0].Idx)
	assert.Equal(t, "EXP", records[0].Diff)

	assert.Equal(t, "2", records[1].Idx)
	assert.Equal(t, "BAS", records[1].Diff)

	assert.Equal(t, "2", records[2].Idx)
	assert.Equal(t, "MAS", records[2].Diff)

	assert.Equal(t, "10", records[3].Idx)
	assert.Equal(t, "BAS", records[3].Diff)
}
