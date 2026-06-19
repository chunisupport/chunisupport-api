package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
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
		DisplayID:      "test123456789012",
		Title:          "テスト楽曲",
		Artist:         "テストアーティスト",
		GenreID:        &genreID,
		BPM:            &bpm,
		Jacket:         &imgURL,
		IsMaxOPUnknown: true,
	}

	notes1Value := 500
	notes2Value := 800
	notes1, err := notes.NewNotes(notes1Value)
	if err != nil {
		require.Failf(t, "前提条件失敗", "notes.NewNotes failed for notes1Value: %v", err)
	}
	notes2, err := notes.NewNotes(notes2Value)
	if err != nil {
		require.Failf(t, "前提条件失敗", "notes.NewNotes failed for notes2Value: %v", err)
	}

	charts := []*entity.Chart{
		{
			DifficultyID:   1, // basic
			Const:          7.5,
			IsConstUnknown: false,
			Notes:          &notes1,
			NotesDesigner:  stringPtr("譜面作者A"),
		},
		{
			DifficultyID:   3, // expert
			Const:          12.0,
			IsConstUnknown: false,
			Notes:          &notes2,
			NotesDesigner:  stringPtr("譜面作者B"),
		},
	}

	song.Charts = charts
	song.OpTargetDifficultyID = 3

	// 変換実行
	dto := handler.convertToSongDTO(song)

	// アサーション
	if dto == nil {
		require.Fail(t, "convertToSongDTO returned nil")
	}

	if dto.DisplayID != "test123456789012" {
		assert.Failf(t, "アサーション失敗", "DisplayID = %v, want %v", dto.DisplayID, "test123456789012")
	}

	if dto.MaxOP != 90 {
		assert.Failf(t, "アサーション失敗", "MaxOP = %v, want %v", dto.MaxOP, 90)
	}

	// IsMaxOPUnknown が反映されていることを確認
	if !dto.IsMaxOPUnknown {
		assert.Failf(t, "アサーション失敗", "IsMaxOPUnknown = %v, want %v", dto.IsMaxOPUnknown, true)
	}

	if dto.OpTargetDifficulty == nil || *dto.OpTargetDifficulty != "EXPERT" {
		assert.Failf(t, "アサーション失敗", "OpTargetDifficulty = %v, want %v", dto.OpTargetDifficulty, "EXPERT")
	}

	// Charts マップのキーが存在するか確認
	if dto.Charts == nil {
		require.Fail(t, "Charts is nil")
	}

	// BASIC 譜面が存在することを確認
	if basicChart, ok := dto.Charts["BASIC"]; !ok || basicChart == nil {
		t.Error("BASIC chart not found")
	} else {
		if basicChart.Const != 7.5 {
			assert.Failf(t, "アサーション失敗", "BASIC chart Const = %v, want %v", basicChart.Const, 7.5)
		}
		if basicChart.NotesDesigner == nil || *basicChart.NotesDesigner != "譜面作者A" {
			assert.Failf(t, "アサーション失敗", "BASIC chart NotesDesigner = %v, want %v", basicChart.NotesDesigner, "譜面作者A")
		}
	}

	// EXPERT 譜面が存在することを確認
	if expertChart, ok := dto.Charts["EXPERT"]; !ok || expertChart == nil {
		t.Error("EXPERT chart not found")
	} else {
		if expertChart.Const != 12.0 {
			assert.Failf(t, "アサーション失敗", "expert chart Const = %v, want %v", expertChart.Const, 12.0)
		}
		if expertChart.NotesDesigner == nil || *expertChart.NotesDesigner != "譜面作者B" {
			assert.Failf(t, "アサーション失敗", "EXPERT chart NotesDesigner = %v, want %v", expertChart.NotesDesigner, "譜面作者B")
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
			body:             `[{"id":"1234567890123456","title":"テスト楽曲","artist":"テストアーティスト","charts":{"BASIC":{"const":10.5,"notes_designer":"譜面作者A"}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*api_internal.UpdateSongRequest) {
				t.Helper()
				if len(requests) != 1 {
					require.Failf(t, "前提条件失敗", "requests len = %d, want 1", len(requests))
				}
				if requests[0] == nil {
					require.Fail(t, "requests[0] should not be nil")
				}
				if requests[0].DisplayID != "1234567890123456" {
					require.Failf(t, "前提条件失敗", "DisplayID = %s, want 1234567890123456", requests[0].DisplayID)
				}
				if len(requests[0].Charts) != 1 {
					require.Failf(t, "前提条件失敗", "Charts len = %d, want 1", len(requests[0].Charts))
				}
				if _, ok := requests[0].Charts["BASIC"]; !ok {
					require.Fail(t, "Charts['BASIC'] should exist")
				}
				if requests[0].Charts["BASIC"].NotesDesigner == nil || *requests[0].Charts["BASIC"].NotesDesigner != "譜面作者A" {
					require.Failf(t, "前提条件失敗", "Charts['BASIC'].NotesDesigner = %v, want %v", requests[0].Charts["BASIC"].NotesDesigner, "譜面作者A")
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
					require.Failf(t, "前提条件失敗", "UpdateSongs returned error: %v", err)
				}
				if rec.Code != tc.expectedStatus {
					require.Failf(t, "前提条件失敗", "Status code = %d, want %d", rec.Code, tc.expectedStatus)
				}
			} else {
				if err == nil {
					require.Fail(t, "UpdateSongs should return error")
				}

				apiErr, ok := err.(*apierror.APIError)
				if !ok {
					require.Failf(t, "前提条件失敗", "error type = %T, want *apierror.APIError", err)
				}
				if apiErr.Code != tc.expectedErrCode {
					require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, tc.expectedErrCode)
				}
				if apiErr.Internal == nil {
					require.Fail(t, "internal error should not be nil")
				}
			}

			if called != tc.expectUsecaseHit {
				require.Failf(t, "前提条件失敗", "UpdateSongs usecase called = %v, want %v", called, tc.expectUsecaseHit)
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
					NotesDesigner:  stringPtr("譜面作者A"),
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
		RatingBands: []*ratingband.RatingBand{},
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
		require.Failf(t, "前提条件失敗", "GetSongs returned error: %v", err)
	}

	// レスポンスの確認
	if rec.Code != http.StatusOK {
		assert.Failf(t, "アサーション失敗", "Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	// レスポンスボディの確認
	var response api_internal.SongsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		require.Failf(t, "前提条件失敗", "Failed to unmarshal response: %v", err)
	}

	if len(response.Songs) != 1 {
		assert.Failf(t, "アサーション失敗", "Songs count = %d, want %d", len(response.Songs), 1)
	}

	if len(response.Songs) > 0 && response.Songs[0].DisplayID != "test123456789012" {
		assert.Failf(t, "アサーション失敗", "DisplayID = %v, want %v", response.Songs[0].DisplayID, "test123456789012")
	}

	// JSONレスポンスの詳細確認
	t.Logf("Response JSON: %s", rec.Body.String())

	// Chartsフィールドの存在確認
	if len(response.Songs) > 0 {
		song := response.Songs[0]
		if song.Charts == nil {
			require.Fail(t, "Charts is nil")
		}

		// 全難易度のキーが存在するか確認
		expectedDiffs := []string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}
		for _, diff := range expectedDiffs {
			if _, exists := song.Charts[diff]; !exists {
				assert.Failf(t, "アサーション失敗", "Charts should contain key '%s'", diff)
			}
		}

		// BASICは存在するはず
		if basicChart := song.Charts["BASIC"]; basicChart == nil {
			t.Error("BASIC chart should not be nil")
		} else if basicChart.NotesDesigner == nil || *basicChart.NotesDesigner != "譜面作者A" {
			assert.Failf(t, "アサーション失敗", "BASIC chart NotesDesigner = %v, want %v", basicChart.NotesDesigner, "譜面作者A")
		}

		// ADVANCED, EXPERT, MASTER, ULTIMAはnullのはず（テストデータにないため）
		if advChart := song.Charts["ADVANCED"]; advChart != nil {
			t.Error("ADVANCED chart should be nil")
		}
	}
}

func TestSongHandler_DeleteSong(t *testing.T) {
	e := echo.New()
	staticMasterCache := &masterdata.StaticCache{}
	masterCache := &masterdata.Cache{}

	t.Run("楽曲が存在しない場合はsong_not_foundを返す", func(t *testing.T) {
		// Given
		mockUsecase := &testutil.MockSongUsecase{
			DeleteSongFunc: func(ctx context.Context, displayID string) error {
				assert.Equal(t, "0000000000000000", displayID)
				return repository.ErrSongNotFound
			},
		}
		handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)
		req := httptest.NewRequest(http.MethodDelete, "/internal/songs/0000000000000000", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("displayid")
		c.SetParamValues("0000000000000000")

		// When
		err := handler.DeleteSong(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeSongNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
	})

	t.Run("不正なDisplayIDの場合はvalidation_failedを返しユースケースを呼ばない", func(t *testing.T) {
		// Given
		called := false
		mockUsecase := &testutil.MockSongUsecase{
			DeleteSongFunc: func(ctx context.Context, displayID string) error {
				called = true
				return nil
			},
		}
		handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)
		req := httptest.NewRequest(http.MethodDelete, "/internal/songs/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("displayid")
		c.SetParamValues("invalid")

		// When
		err := handler.DeleteSong(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
		}
		assert.False(t, called)
	})
}

func TestSongHandler_RestoreSong(t *testing.T) {
	e := echo.New()
	staticMasterCache := &masterdata.StaticCache{}
	masterCache := &masterdata.Cache{}

	t.Run("楽曲が存在しない場合はsong_not_foundを返す", func(t *testing.T) {
		// Given
		mockUsecase := &testutil.MockSongUsecase{
			RestoreSongFunc: func(ctx context.Context, displayID string) error {
				assert.Equal(t, "0000000000000000", displayID)
				return repository.ErrSongNotFound
			},
		}
		handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)
		req := httptest.NewRequest(http.MethodPost, "/internal/songs/0000000000000000/restore", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("displayid")
		c.SetParamValues("0000000000000000")

		// When
		err := handler.RestoreSong(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeSongNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
	})
}

func stringPtr(value string) *string {
	return &value
}
