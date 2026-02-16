package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type testValidator struct {
	validator *validator.Validate
}

func (tv *testValidator) Validate(i any) error {
	// validator.v10のStruct()はスライスを直接サポートしないため、
	// このテスト用バリデータは構造体のみを対象とします。
	// スライスのバリデーションはハンドラ側でループ処理する必要があります。
	return tv.validator.Struct(i)
}

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
		songUsecase: &testutil.MockSongUsecase{},
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

	if dto.MaxOP != 90 {
		t.Errorf("MaxOP = %v, want %v", dto.MaxOP, 90)
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

// TestUpdateSongs はUpdateSongsハンドラーの入力バリデーションをテストします。
func TestUpdateSongs(t *testing.T) {
	masterCache := &masterdata.Cache{}
	staticMasterCache := &masterdata.StaticCache{}

	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newPutSongsContext := func(body string) echo.Context {
		req := httptest.NewRequest(http.MethodPut, "/internal/songs", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}
	testCases := []struct {
		name             string
		body             string
		expectedStatus   int
		expectedErrCode  string
		expectUsecaseHit bool
		assertUsecaseReq func(t *testing.T, requests []*api_internal.UpdateSongRequest)
	}{
		{
			name:             "正常な配列で204が返る",
			body:             `[{"id":"1234567890123456","title":"テスト楽曲","artist":"テストアーティスト","charts":{"BASIC":{"const":10.5}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*api_internal.UpdateSongRequest) {
				t.Helper()
				if len(requests) != 1 {
					t.Fatalf("requests len = %d, want 1", len(requests))
				}
				if requests[0] == nil {
					t.Fatal("requests[0] should not be nil")
				}
				if requests[0].DisplayID != "1234567890123456" {
					t.Fatalf("DisplayID = %s, want 1234567890123456", requests[0].DisplayID)
				}
				if len(requests[0].Charts) != 1 {
					t.Fatalf("Charts len = %d, want 1", len(requests[0].Charts))
				}
				if _, ok := requests[0].Charts["BASIC"]; !ok {
					t.Fatal("Charts['BASIC'] should exist")
				}
			},
		},
		{
			name:            "不正要素を含む配列でvalidation_failedが返る",
			body:            `[{"id":"short","title":"テスト楽曲","artist":"テストアーティスト"}]`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
		{
			name:            "null要素を含む配列でvalidation_failedが返る",
			body:            `[null]`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
		{
			name:            "トップレベルnullでvalidation_failedが返る",
			body:            `null`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
		{
			name:            "chartsにnull要素を含む配列でvalidation_failedが返る",
			body:            `[{"id":"1234567890123456","title":"テスト楽曲","artist":"テストアーティスト","charts":{"BASIC":null}}]`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockUsecase := &testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
					called = true
					if tc.assertUsecaseReq != nil {
						tc.assertUsecaseReq(t, requests)
					}
					return nil
				},
			}
			handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)

			c := newPutSongsContext(tc.body)
			rec := c.Response().Writer.(*httptest.ResponseRecorder)

			err := handler.UpdateSongs(c)

			if tc.expectedErrCode == "" {
				if err != nil {
					t.Fatalf("UpdateSongs returned error: %v", err)
				}
				if rec.Code != tc.expectedStatus {
					t.Fatalf("Status code = %d, want %d", rec.Code, tc.expectedStatus)
				}
			} else {
				if err == nil {
					t.Fatal("UpdateSongs should return error")
				}

				apiErr, ok := err.(*apierror.APIError)
				if !ok {
					t.Fatalf("error type = %T, want *apierror.APIError", err)
				}
				if apiErr.Code != tc.expectedErrCode {
					t.Fatalf("api error code = %s, want %s", apiErr.Code, tc.expectedErrCode)
				}
				if apiErr.Internal == nil {
					t.Fatal("internal error should not be nil")
				}
			}

			if called != tc.expectUsecaseHit {
				t.Fatalf("UpdateSongs usecase called = %v, want %v", called, tc.expectUsecaseHit)
			}
		})
	}

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
	mockUsecase := &testutil.MockSongUsecase{
		GetAllSongsExcludingWorldsendFunc: func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error) {
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
