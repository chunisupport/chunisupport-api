package api_v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	api_v1_dto "github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

func TestConvertToV1SongDTO(t *testing.T) {
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

	handler := &V1SongHandler{
		songUsecase: &testutil.MockSongUsecase{},
		masterCache: masterCache,
	}

	genreID := 2
	bpm := 200
	imgURL := "https://example.com/v1jacket.jpg"

	song := &entity.Song{
		DisplayID:      "v1test1234567890",
		Title:          "V1テスト楽曲",
		Artist:         "V1アーティスト",
		GenreID:        &genreID,
		BPM:            &bpm,
		Jacket:         &imgURL,
		IsMaxOPUnknown: true,
	}

	notes1, err := notes.NewNotes(600)
	if err != nil {
		t.Fatalf("notes.NewNotes failed for notes1: %v", err)
	}
	notes2, err := notes.NewNotes(1200)
	if err != nil {
		t.Fatalf("notes.NewNotes failed for notes2: %v", err)
	}

	song.Charts = []*entity.Chart{
		{
			DifficultyID:   2,
			Const:          9.0,
			IsConstUnknown: false,
			Notes:          &notes1,
		},
		{
			DifficultyID:   4,
			Const:          13.7,
			IsConstUnknown: false,
			Notes:          &notes2,
		},
	}

	dto := handler.convertToV1SongDTO(song)

	if dto == nil {
		t.Fatal("convertToV1SongDTO returned nil")
	}
	if dto.DisplayID != "v1test1234567890" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "v1test1234567890")
	}
	if dto.MaxOP != 90 {
		t.Errorf("MaxOP = %v, want %v", dto.MaxOP, 90)
	}
	if !dto.IsMaxOPUnknown {
		t.Errorf("IsMaxOPUnknown = %v, want %v", dto.IsMaxOPUnknown, true)
	}
	if dto.Charts == nil {
		t.Fatal("Charts is nil")
	}
	if advancedChart, ok := dto.Charts["ADVANCED"]; !ok || advancedChart == nil {
		t.Error("ADVANCED chart not found")
	} else if advancedChart.Const != 9.0 {
		t.Errorf("ADVANCED chart Const = %v, want %v", advancedChart.Const, 9.0)
	}
	if masterChart, ok := dto.Charts["MASTER"]; !ok || masterChart == nil {
		t.Error("MASTER chart not found")
	} else if masterChart.Const != 13.7 {
		t.Errorf("MASTER chart Const = %v, want %v", masterChart.Const, 13.7)
	}
	if basicChart, ok := dto.Charts["BASIC"]; !ok {
		t.Error("BASIC key not found in map")
	} else if basicChart != nil {
		t.Error("BASIC chart should be nil")
	}
	if expertChart, ok := dto.Charts["EXPERT"]; !ok {
		t.Error("EXPERT key not found in map")
	} else if expertChart != nil {
		t.Error("EXPERT chart should be nil")
	}
	if ultimaChart, ok := dto.Charts["ULTIMA"]; !ok {
		t.Error("ULTIMA key not found in map")
	} else if ultimaChart != nil {
		t.Error("ULTIMA chart should be nil")
	}
}

func TestGetSongs(t *testing.T) {
	masterCache := &masterdata.Cache{
		GenreNamesByID:      map[int]string{1: "POPS & ANIME"},
		DifficultyNamesByID: map[int]string{1: "BASIC"},
	}
	updatedAt := time.Date(2026, 3, 22, 15, 4, 5, 0, time.UTC)

	handler := &V1SongHandler{
		songUsecase: &testutil.MockSongUsecase{
			GetAllSongsExcludingWorldsendFunc: func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) (*usecase.SongListResult, error) {
				return &usecase.SongListResult{
					Songs:     []*entity.Song{{DisplayID: "v1songs123456789", Title: "楽曲", Artist: "アーティスト", Charts: []*entity.Chart{}}},
					UpdatedAt: &updatedAt,
				}, nil
			},
		},
		masterCache: masterCache,
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetSongs(c)
	if err != nil {
		t.Fatalf("GetSongs returned error: %v", err)
	}

	var response api_v1_dto.V1SongsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.UpdatedAt == nil {
		t.Fatal("UpdatedAt should not be nil")
	}
}
