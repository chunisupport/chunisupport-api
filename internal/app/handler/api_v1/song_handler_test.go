package api_v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/notes"
	"github.com/Qman110101/chunisupport-api/internal/dto"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v4"
)

// TestConvertToV1SongDTO はV1SongHandlerのconvertToV1SongDTOメソッドをテストします。
func TestConvertToV1SongDTO(t *testing.T) {
	// マスタデータキャッシュの準備
	masterCache := &masterdata.Cache{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
			2: "niconico",
		},
		DifficultyNamesByID: map[int]string{
			1: "basic",
			2: "advanced",
			3: "expert",
			4: "master",
			5: "ultima",
		},
	}

	handler := &V1SongHandler{
		masterCache: masterCache,
	}

	// テストデータの準備
	genreID := 2
	bpm := 200
	imgURL := "https://example.com/v1jacket.jpg"

	song := &entity.Song{
		DisplayID: "v1test1234567890",
		Title:     "V1テスト楽曲",
		Artist:    "V1アーティスト",
		GenreID:   &genreID,
		BPM:       &bpm,
		Jacket:    &imgURL,
	}

	notes1Value := 600
	notes2Value := 1200
	notes1, err := notes.NewNotes(notes1Value)
	if err != nil {
		t.Fatalf("notes.NewNotes failed for notes1Value: %v", err)
	}
	notes2, err := notes.NewNotes(notes2Value)
	if err != nil {
		t.Fatalf("notes.NewNotes failed for notes2Value: %v", err)
	}

	charts := []*entity.Chart{
		{
			DifficultyID:   2, // advanced
			Const:          9.0,
			IsConstUnknown: false,
			Notes:          &notes1,
		},
		{
			DifficultyID:   4, // master
			Const:          13.7,
			IsConstUnknown: false,
			Notes:          &notes2,
		},
	}

	swc := &repository.SongWithCharts{
		Song:   song,
		Charts: charts,
	}

	// 変換実行
	dto := handler.convertToV1SongDTO(swc)

	// アサーション
	if dto == nil {
		t.Fatal("convertToV1SongDTO returned nil")
	}

	if dto.DisplayID != "v1test1234567890" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "v1test1234567890")
	}

	// Charts マップのキーが存在するか確認
	if dto.Charts == nil {
		t.Fatal("Charts is nil")
	}

	// advanced 譜面が存在することを確認
	if advancedChart, ok := dto.Charts["advanced"]; !ok || advancedChart == nil {
		t.Error("advanced chart not found")
	} else {
		if advancedChart.Const != 9.0 {
			t.Errorf("advanced chart Const = %v, want %v", advancedChart.Const, 9.0)
		}
	}

	// master 譜面が存在することを確認
	if masterChart, ok := dto.Charts["master"]; !ok || masterChart == nil {
		t.Error("master chart not found")
	} else {
		if masterChart.Const != 13.7 {
			t.Errorf("master chart Const = %v, want %v", masterChart.Const, 13.7)
		}
	}

	// basic 譜面は存在しないので nil であることを確認
	if basicChart, ok := dto.Charts["basic"]; !ok {
		t.Error("basic key not found in map")
	} else if basicChart != nil {
		t.Error("basic chart should be nil")
	}

	// expert 譜面は存在しないので nil であることを確認
	if expertChart, ok := dto.Charts["expert"]; !ok {
		t.Error("expert key not found in map")
	} else if expertChart != nil {
		t.Error("expert chart should be nil")
	}

	// ultima 譜面は存在しないので nil であることを確認
	if ultimaChart, ok := dto.Charts["ultima"]; !ok {
		t.Error("ultima key not found in map")
	} else if ultimaChart != nil {
		t.Error("ultima chart should be nil")
	}
}

func TestGetSongStats(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/songs/test123456789012/stat", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("displayid")
	c.SetParamValues("test123456789012")

	staticMasterCache := &masterdata.StaticCache{
		RatingBands: []*entity.RatingBand{},
	}

	handler := &V1SongHandler{
		statsUsecase: &testutil.MockChartStatsUsecase{
			Stats: &entity.SongChartStats{
				SongID: "test123456789012",
				Charts: map[string][]*entity.ChartStatsByRatingBand{
					"MASTER": {},
				},
			},
		},
		staticMasterCache: staticMasterCache,
	}

	if err := handler.GetSongStats(c); err != nil {
		t.Fatalf("GetSongStats returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response dto.ChartStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.SongID != "test123456789012" {
		t.Errorf("SongID = %v, want %v", response.SongID, "test123456789012")
	}
}
