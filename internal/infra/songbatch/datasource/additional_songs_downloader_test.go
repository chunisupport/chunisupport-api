package datasource

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdditionalSongsParsersKeepEmptyResultsAsJSONArrays(t *testing.T) {
	tests := []struct {
		name      string
		songRows  [][]string
		chartRows [][]string
		weRows    [][]string
	}{
		{
			name:      "header only",
			songRows:  [][]string{{"id", "title", "artist", "genre", "release"}},
			chartRows: [][]string{{"id", "diff", "const"}},
			weRows:    [][]string{{"id", "title", "artist", "genre", "release"}},
		},
		{
			name:      "all data rows are skipped",
			songRows:  [][]string{{"id", "title", "artist", "genre", "release"}, {"", "invalid"}},
			chartRows: [][]string{{"id", "diff", "const"}, {"", "", ""}},
			weRows:    [][]string{{"id", "title", "artist", "genre", "release"}, {"", "invalid"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AdditionalSongsDownloader{}
			songs, err := d.parseSongsSheet(tt.songRows)
			if err != nil {
				t.Fatalf("parseSongsSheet() error = %v", err)
			}
			charts, err := d.parseChartsSheet(tt.chartRows)
			if err != nil {
				t.Fatalf("parseChartsSheet() error = %v", err)
			}
			weCharts, err := d.parseWEChartsSheet(tt.weRows)
			if err != nil {
				t.Fatalf("parseWEChartsSheet() error = %v", err)
			}
			if songs == nil || charts == nil || weCharts == nil {
				t.Fatalf("empty parser results must be non-nil: songs=%v charts=%v weCharts=%v", songs, charts, weCharts)
			}

			data, err := json.Marshal(additionalSongsSheetData{
				Songs:    songs,
				Charts:   charts,
				WECharts: weCharts,
				Courses:  []additionalCourseRow{},
			})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			const want = `{"songs":[],"charts":[],"we_charts":[],"courses":[]}`
			if string(data) != want {
				t.Fatalf("empty results JSON = %s, want %s", data, want)
			}
		})
	}
}

func TestParseCoursesSheet(t *testing.T) {
	d := &AdditionalSongsDownloader{}
	values := [][]string{
		{"id", "title", "class"},
		{"50000", " First Step ", "1"},
		{"50020", "HORIZON Set", "inf"},
		{"50027", "RANDOM", "extra"},
	}

	courses, err := d.parseCoursesSheet(values)
	if err != nil {
		t.Fatalf("parseCoursesSheet() error = %v", err)
	}
	if len(courses) != 3 {
		t.Fatalf("len(courses) = %d, want 3", len(courses))
	}
	if courses[0].ID != "50000" || courses[0].Title != "First Step" || courses[0].Class != "1" {
		t.Errorf("courses[0] = %+v", courses[0])
	}
	if courses[2].Class != "extra" {
		t.Errorf("courses[2].Class = %q, want extra", courses[2].Class)
	}
}

// TestParseSongsSheet_ValidRow は additional_songs の有効行1件が全列で正しく変換されることを検証します。
// 本番シート列順（id〜imgの22列）に合わせ、4難易度定数あり・ULTIMAなし（0→nilの代表値）を固定値比較します。
func TestParseSongsSheet_ValidRow(t *testing.T) {
	d := &AdditionalSongsDownloader{}
	values := [][]string{
		{"id", "title", "artist", "genre", "release", "bpm", "basnt", "bas", "basuk", "advnt", "adv", "advuk", "expnt", "exp", "expuk", "masnt", "mas", "masuk", "ultnt", "ult", "ultuk", "img"},
		{"10001", "Test Song", "Test Artist", "ORIGINAL", "2024/01/01", "180", "500", "3.0", "", "800", "6.0", "", "1200", "9.5", "", "1800", "12.5", "1", "0", "0", "", "abc123"},
	}

	songs, err := d.parseSongsSheet(values)
	require.NoError(t, err)
	require.Len(t, songs, 1)

	want := additionalSongRow{
		ID:      "10001",
		Title:   "Test Song",
		Artist:  "Test Artist",
		Genre:   "ORIGINAL",
		Release: "2024/01/01",
		BPM:     new(180),
		BasNt:   new(500),
		Bas:     new(3.0),
		BasUK:   false,
		AdvNt:   new(800),
		Adv:     new(6.0),
		AdvUK:   false,
		ExpNt:   new(1200),
		Exp:     new(9.5),
		ExpUK:   false,
		MasNt:   new(1800),
		Mas:     new(12.5),
		MasUK:   true,
		UltNt:   nil,
		Ult:     nil,
		UltUK:   false,
		Img:     "abc123",
	}
	assert.Equal(t, want, songs[0])
}

// TestParseSongsSheet_SkipsRowWithMissingConstant は4難易度のうち1つでも欠けると行全体が採用されないことを検証します。
// 既存の空結果テストと重複させず、代表としてEXPERT定数欠落の1件だけを確認します。
func TestParseSongsSheet_SkipsRowWithMissingConstant(t *testing.T) {
	d := &AdditionalSongsDownloader{}
	values := [][]string{
		{"id", "title", "artist", "genre", "release", "bpm", "basnt", "bas", "basuk", "advnt", "adv", "advuk", "expnt", "exp", "expuk", "masnt", "mas", "masuk", "ultnt", "ult", "ultuk", "img"},
		{"10001", "Test Song", "Test Artist", "ORIGINAL", "2024/01/01", "180", "500", "3.0", "", "800", "6.0", "", "1200", "", "", "1800", "12.5", "", "", "", "", "abc123"},
	}

	songs, err := d.parseSongsSheet(values)
	require.NoError(t, err)
	assert.Empty(t, songs)
	assert.NotNil(t, songs)
}

// TestParseChartsSheet_ValidRow は additional_charts の有効行1件が全列で正しく変換されることを検証します。
// 本番シート列順（id, diff, const, csuk, notes）に合わせ、notesの0→nil代表値を含めて固定値比較します。
func TestParseChartsSheet_ValidRow(t *testing.T) {
	d := &AdditionalSongsDownloader{}
	values := [][]string{
		{"id", "diff", "const", "csuk", "notes"},
		{"10001", "ULTIMA", "13.5", "true", "0"},
	}

	charts, err := d.parseChartsSheet(values)
	require.NoError(t, err)
	require.Len(t, charts, 1)

	want := additionalChartRow{
		ID:    "10001",
		Diff:  "ULTIMA",
		Const: new(13.5),
		CsUK:  true,
		Notes: nil,
	}
	assert.Equal(t, want, charts[0])
}

// TestParseWEChartsSheet_ValidRow は additional_songs_charts_we の有効行1件が全列で正しく変換されることを検証します。
// 本番シート列順（id〜imgの9列）に合わせ、notesの0→nil代表値を含めて固定値比較します。
func TestParseWEChartsSheet_ValidRow(t *testing.T) {
	d := &AdditionalSongsDownloader{}
	values := [][]string{
		{"id", "title", "artist", "genre", "release", "we_kanji", "we_star", "notes", "img"},
		{"we001", "WE Song", "WE Artist", "ORIGINAL", "2024/02/01", "光", "5", "0", "def456"},
	}

	weCharts, err := d.parseWEChartsSheet(values)
	require.NoError(t, err)
	require.Len(t, weCharts, 1)

	want := additionalWEChartRow{
		ID:      "we001",
		Title:   "WE Song",
		Artist:  "WE Artist",
		Genre:   "ORIGINAL",
		Release: "2024/02/01",
		WEKanji: "光",
		WEStar:  new(5),
		Notes:   nil,
		Img:     "def456",
	}
	assert.Equal(t, want, weCharts[0])
}

func TestParseCoursesSheetRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name    string
		values  [][]string
		wantErr string
	}{
		{
			name:    "必須項目不足",
			values:  [][]string{{"id", "title", "class"}, {"50000", "First Step"}},
			wantErr: "missing required fields",
		},
		{
			name:    "ID重複",
			values:  [][]string{{"id", "title", "class"}, {"50000", "First Step", "1"}, {"50000", "Other", "2"}},
			wantErr: "duplicate id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AdditionalSongsDownloader{}
			_, err := d.parseCoursesSheet(tt.values)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
