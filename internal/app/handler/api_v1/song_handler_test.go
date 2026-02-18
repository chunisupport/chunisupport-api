package api_v1

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
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

	// テストデータの準備
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

	song.Charts = charts

	// 変換実行
	dto := handler.convertToV1SongDTO(song)

	// アサーション
	if dto == nil {
		t.Fatal("convertToV1SongDTO returned nil")
	}

	if dto.DisplayID != "v1test1234567890" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "v1test1234567890")
	}

	if dto.MaxOP != 90 {
		t.Errorf("MaxOP = %v, want %v", dto.MaxOP, 90)
	}

	// IsMaxOPUnknown が反映されていることを確認
	if !dto.IsMaxOPUnknown {
		t.Errorf("IsMaxOPUnknown = %v, want %v", dto.IsMaxOPUnknown, true)
	}

	// Charts マップのキーが存在するか確認
	if dto.Charts == nil {
		t.Fatal("Charts is nil")
	}

	// advanced 譜面が存在することを確認
	if advancedChart, ok := dto.Charts["ADVANCED"]; !ok || advancedChart == nil {
		t.Error("ADVANCED chart not found")
	} else {
		if advancedChart.Const != 9.0 {
			t.Errorf("ADVANCED chart Const = %v, want %v", advancedChart.Const, 9.0)
		}
	}

	// master 譜面が存在することを確認
	if masterChart, ok := dto.Charts["MASTER"]; !ok || masterChart == nil {
		t.Error("MASTER chart not found")
	} else {
		if masterChart.Const != 13.7 {
			t.Errorf("MASTER chart Const = %v, want %v", masterChart.Const, 13.7)
		}
	}

	// basic 譜面は存在しないので nil であることを確認
	if basicChart, ok := dto.Charts["BASIC"]; !ok {
		t.Error("BASIC key not found in map")
	} else if basicChart != nil {
		t.Error("BASIC chart should be nil")
	}

	// expert 譜面は存在しないので nil であることを確認
	if expertChart, ok := dto.Charts["EXPERT"]; !ok {
		t.Error("EXPERT key not found in map")
	} else if expertChart != nil {
		t.Error("EXPERT chart should be nil")
	}

	// ultima 譜面は存在しないので nil であることを確認
	if ultimaChart, ok := dto.Charts["ULTIMA"]; !ok {
		t.Error("ULTIMA key not found in map")
	} else if ultimaChart != nil {
		t.Error("ULTIMA chart should be nil")
	}
}
