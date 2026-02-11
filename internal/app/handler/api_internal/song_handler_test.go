package api_internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v4"
)

// TestConvertToSongDTO はSongHandlerのconvertToSongDTOメソッドをテストします。
func TestConvertToSongDTO(t *testing.T) {
	// マスタデータキャッシュの準備
	masterCache := &masterdata.Cache{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
			2: "niconico",
		},
		DifficultyNamesByID: map[int]string{
			1: "BASIC",
			2: "ADVANCED",
			3: "EXPERT",
			4: "MASTER",
			5: "ULTIMA",
		},
	}

	handler := &SongHandler{
		songUsecase: &mockSongUsecase{},
		masterCache: masterCache,
	}

	// テストデータの準備
	genreID := 1
	bpm := 180
	imgURL := "https://example.com/jacket.jpg"

	song := &entity.Song{
		DisplayID: "test123456789012",
		Title:     "テスト楽曲",
		Artist:    "テストアーティスト",
		GenreID:   &genreID,
		BPM:       &bpm,
		Jacket:    &imgURL,
	}

	notes1Value := 500
	notes2Value := 800
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
			DifficultyID:   1, // basic
			Const:          7.5,
			IsConstUnknown: false,
			Notes:          &notes1,
		},
		{
			DifficultyID:   3, // expert
			Const:          12.0,
			IsConstUnknown: false,
			Notes:          &notes2,
		},
	}

	song.Charts = charts

	// 変換実行
	dto := handler.convertToSongDTO(song)

	// アサーション
	if dto == nil {
		t.Fatal("convertToSongDTO returned nil")
	}

	if dto.DisplayID != "test123456789012" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "test123456789012")
	}

	// Charts マップのキーが存在するか確認
	if dto.Charts == nil {
		t.Fatal("Charts is nil")
	}

	// BASIC 譜面が存在することを確認
	if basicChart, ok := dto.Charts["BASIC"]; !ok || basicChart == nil {
		t.Error("BASIC chart not found")
	} else {
		if basicChart.Const != 7.5 {
			t.Errorf("BASIC chart Const = %v, want %v", basicChart.Const, 7.5)
		}
	}

	// EXPERT 譜面が存在することを確認
	if expertChart, ok := dto.Charts["EXPERT"]; !ok || expertChart == nil {
		t.Error("EXPERT chart not found")
	} else {
		if expertChart.Const != 12.0 {
			t.Errorf("expert chart Const = %v, want %v", expertChart.Const, 12.0)
		}
	}

	// ADVANCED 譜面は存在しないので nil であることを確認
	if advancedChart, ok := dto.Charts["ADVANCED"]; !ok {
		t.Error("ADVANCED key not found in map")
	} else if advancedChart != nil {
		t.Error("ADVANCED chart should be nil")
	}

	// MASTER 譜面は存在しないので nil であることを確認
	if masterChart, ok := dto.Charts["MASTER"]; !ok {
		t.Error("MASTER key not found in map")
	} else if masterChart != nil {
		t.Error("MASTER chart should be nil")
	}

	// ULTIMA 譜面は存在しないので nil であることを確認
	if ultimaChart, ok := dto.Charts["ULTIMA"]; !ok {
		t.Error("ULTIMA key not found in map")
	} else if ultimaChart != nil {
		t.Error("ultima chart should be nil")
	}
}

// mockSongUsecase はSongUsecaseのモック実装です。
type mockSongUsecase struct {
	getAllSongsFunc func(ctx context.Context, includeDeleted bool) ([]*entity.Song, error)
}

func (m *mockSongUsecase) GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool) ([]*entity.Song, error) {
	if m.getAllSongsFunc != nil {
		return m.getAllSongsFunc(ctx, includeDeleted)
	}
	return nil, nil
}

func (m *mockSongUsecase) GetSongByDisplayID(ctx context.Context, displayID string) (*entity.Song, error) {
	return nil, nil
}

func (m *mockSongUsecase) DeleteSong(ctx context.Context, displayID string) error {
	return nil
}

func (m *mockSongUsecase) RestoreSong(ctx context.Context, displayID string) error {
	return nil
}

func (m *mockSongUsecase) UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
	return nil
}

func (m *mockSongUsecase) CalcSongMaxOP(song *entity.Song) float64 {
	if song == nil {
		return 0
	}
	return 90
}

// TestGetSongs はGetSongsハンドラーの基本動作をテストします。
func TestGetSongs(t *testing.T) {
	// マスタデータキャッシュの準備
	masterCache := &masterdata.Cache{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
		},
		DifficultyNamesByID: map[int]string{
			1: "BASIC",
			4: "MASTER",
		},
	}

	// テストデータの準備
	genreID := 1
	bpm := 180
	notes1Value := 500
	notes1, _ := notes.NewNotes(notes1Value)

	testSongs := []*entity.Song{
		{
			DisplayID: "test123456789012",
			Title:     "テスト楽曲",
			Artist:    "テストアーティスト",
			GenreID:   &genreID,
			BPM:       &bpm,
			Charts: []*entity.Chart{
				{
					DifficultyID:   1,
					Const:          7.5,
					IsConstUnknown: false,
					Notes:          &notes1,
				},
			},
		},
	}

	// モックUsecaseの準備
	mockUsecase := &mockSongUsecase{
		getAllSongsFunc: func(ctx context.Context, includeDeleted bool) ([]*entity.Song, error) {
			return testSongs, nil
		},
	}

	// ハンドラーの準備
	staticMasterCache := &masterdata.StaticCache{
		RatingBands: []*entity.RatingBand{},
	}
	handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)

	// リクエストの作成
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// テスト実行
	err := handler.GetSongs(c)
	if err != nil {
		t.Fatalf("GetSongs returned error: %v", err)
	}

	// レスポンスの確認
	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	// レスポンスボディの確認
	var response api_internal.SongsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Songs) != 1 {
		t.Errorf("Songs count = %d, want %d", len(response.Songs), 1)
	}

	if len(response.Songs) > 0 && response.Songs[0].DisplayID != "test123456789012" {
		t.Errorf("DisplayID = %v, want %v", response.Songs[0].DisplayID, "test123456789012")
	}

	// JSONレスポンスの詳細確認
	t.Logf("Response JSON: %s", rec.Body.String())

	// Chartsフィールドの存在確認
	if len(response.Songs) > 0 {
		song := response.Songs[0]
		if song.Charts == nil {
			t.Fatal("Charts is nil")
		}

		// 全難易度のキーが存在するか確認
		expectedDiffs := []string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}
		for _, diff := range expectedDiffs {
			if _, exists := song.Charts[diff]; !exists {
				t.Errorf("Charts should contain key '%s'", diff)
			}
		}

		// BASICは存在するはず
		if basicChart := song.Charts["BASIC"]; basicChart == nil {
			t.Error("BASIC chart should not be nil")
		}

		// ADVANCED, EXPERT, MASTER, ULTIMAはnullのはず（テストデータにないため）
		if advChart := song.Charts["ADVANCED"]; advChart != nil {
			t.Error("ADVANCED chart should be nil")
		}
	}
}
