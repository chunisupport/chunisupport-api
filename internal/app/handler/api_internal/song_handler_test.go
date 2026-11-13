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
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestUpdateSongs はUpdateSongsハンドラーの入力バリデーションをテストします。
func TestUpdateSongs(t *testing.T) {
	masterCache := &masterdata.Cache{}
	staticMasterCache := &masterdata.StaticCache{}

	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newPutSongsContext := func(body string) *echo.Context {
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
		assertUsecaseReq func(t *testing.T, requests []*usecase.UpdateSongInput)
	}{
		{
			name:             "正常な配列で204が返る",
			body:             `[{"id":"1234567890123456","title":"テスト楽曲","artist":"テストアーティスト","charts":{"BASIC":{"const":10.5,"notes_designer":"譜面作者A"}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*usecase.UpdateSongInput) {
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
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					called = true
					if tc.assertUsecaseReq != nil {
						tc.assertUsecaseReq(t, requests)
					}
					return nil
				},
			}
			handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)

			c := newPutSongsContext(tc.body)
			response, _ := echo.UnwrapResponse(c.Response())
			rec := response.ResponseWriter.(*httptest.ResponseRecorder)

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

// TestGetSongs は公開の楽曲一覧ハンドラーがDTO契約どおりの固定応答を返すことを検証します。
func TestGetSongs(t *testing.T) {
	masterCache := &masterdata.Cache{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
		},
		DifficultyNamesByID: map[int]string{
			1: "BASIC",
			2: "ADVANCED",
			3: "EXPERT",
			4: "MASTER",
			5: "ULTIMA",
		},
	}

	genreID := 1
	bpm := 180
	notes1Value := 500
	notes1, _ := notes.NewNotes(notes1Value)
	designerExpert := "譜面作者"

	testSongs := []*entity.Song{
		{
			DisplayID: "test123456789012",
			Title:     "テスト楽曲",
			Artist:    "テストアーティスト",
			GenreID:   &genreID,
			BPM:       &bpm,
			IsNew:     true,
			Charts: []*entity.Chart{
				{
					DifficultyID:   1,
					Const:          7.5,
					IsConstUnknown: false,
					Notes:          &notes1,
					NotesDesigner:  new("譜面作者A"),
				},
				{
					DifficultyID:   3,
					Const:          12.0,
					IsConstUnknown: false,
					NotesDesigner:  &designerExpert,
				},
			},
		},
	}

	// Given: 匿名リクエストのためコンテキストに利用者を設定しない。
	var gotIncludeDeleted bool
	var gotRequesterAccountTypeID *int
	mockUsecase := &testutil.MockSongUsecase{
		GetAllSongsExcludingWorldsendFunc: func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error) {
			// usecase へ渡る引数を後で検証できるよう記録する。
			gotIncludeDeleted = includeDeleted
			gotRequesterAccountTypeID = requesterAccountTypeID
			return testSongs, nil
		},
	}

	staticMasterCache := &masterdata.StaticCache{
		RatingBands: []*ratingband.RatingBand{},
	}
	handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetSongs(c)
	if err != nil {
		require.Failf(t, "前提条件失敗", "GetSongs returned error: %v", err)
	}

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

	if len(response.Songs) > 0 {
		assert.True(t, response.Songs[0].IsNew)
	}

	// JSONレスポンスの詳細確認
	t.Logf("Response JSON: %s", rec.Body.String())

	// Chartsフィールドの存在確認
	if len(response.Songs) > 0 {
		song := response.Songs[0]
		if song.Charts == nil {
			require.Fail(t, "Charts is nil")
		}

		// 全難易度のキーが存在するか確認（入力・出力とも大文字難易度で統一）
		expectedDiffs := []string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}
		for _, diff := range expectedDiffs {
			if _, exists := song.Charts[diff]; !exists {
				assert.Failf(t, "アサーション失敗", "Charts should contain key '%s'", diff)
			}
		}

		// 譜面対応と代表値を固定値で検証する。MaxOPはモック既定値の90になる。
		assert.Equal(t, 90.0, song.MaxOP)

		// BASICは存在するはず
		require.NotNil(t, song.Charts["BASIC"], "BASIC譜面が存在すること")
		assert.Equal(t, 7.5, song.Charts["BASIC"].Const.Float64())
		require.NotNil(t, song.Charts["BASIC"].NotesDesigner, "BASIC譜面の作者が設定されていること")
		assert.Equal(t, "譜面作者A", *song.Charts["BASIC"].NotesDesigner)

		// EXPERTは存在するはず
		require.NotNil(t, song.Charts["EXPERT"], "EXPERT譜面が存在すること")
		assert.Equal(t, 12.0, song.Charts["EXPERT"].Const.Float64())
		require.NotNil(t, song.Charts["EXPERT"].NotesDesigner, "EXPERT譜面の作者が設定されていること")
		assert.Equal(t, "譜面作者", *song.Charts["EXPERT"].NotesDesigner)

		// ADVANCED, MASTER, ULTIMAはnullのはず（テストデータにないため）
		assert.Nil(t, song.Charts["ADVANCED"], "ADVANCED譜面は存在しないこと")
		assert.Nil(t, song.Charts["MASTER"], "MASTER譜面は存在しないこと")
		assert.Nil(t, song.Charts["ULTIMA"], "ULTIMA譜面は存在しないこと")

		// 応答全体を固定JSONと比較する（docsのGET /internal/songsの形式に照合）。
		assert.JSONEq(t, `{"songs":[{"id":"test123456789012","title":"テスト楽曲","wiki_page_title":null,"reading":null,"name_folder_code":"","artist":"テストアーティスト","genre":"POPS & ANIME","bpm":180,"release":null,"jacket":null,"official_idx":"","maxop":90,"is_maxop_unknown":false,"op_target_difficulty":null,"is_new":true,"unlock_required":false,"charts":{"BASIC":{"const":7.5,"is_const_unknown":false,"notes":500,"notes_designer":"譜面作者A"},"ADVANCED":null,"EXPERT":{"const":12,"is_const_unknown":false,"notes":null,"notes_designer":"譜面作者"},"MASTER":null,"ULTIMA":null}}]}`, rec.Body.String())
	}

	// Then: 匿名のため include_deleted は既定値 false、権限は nil がそのまま渡る。
	assert.False(t, gotIncludeDeleted)
	assert.Nil(t, gotRequesterAccountTypeID)
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
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "0000000000000000"}})

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
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})

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
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "0000000000000000"}})

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

func TestSongHandler_CreateSong_不正JSONは400(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "Content-Typeなし", body: `{"official_idx":"1","title":"曲","artist":"A","genre":"POPS & ANIME"}`},
		{name: "未知フィールド", contentType: echo.MIMEApplicationJSON, body: `{"official_idx":"1","title":"曲","artist":"A","genre":"POPS & ANIME","unknown":1}`},
		{name: "複数JSON値", contentType: echo.MIMEApplicationJSON, body: `{"official_idx":"1","title":"曲","artist":"A","genre":"POPS & ANIME"} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := NewSongHandler(&testutil.MockSongUsecase{
				CreateSongFunc: func(ctx context.Context, input *usecase.CreateSongInput) (*entity.Song, error) {
					called = true
					return nil, nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			req := httptest.NewRequest(http.MethodPost, "/internal/songs", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}

			err := handler.CreateSong(e.NewContext(req, httptest.NewRecorder()))

			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
			assert.False(t, called)
		})
	}
}

func TestSongHandler_UpdateSongs_不正JSONは400(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "Content-Typeなし", body: `[{"id":"1234567890123456","title":"曲","artist":"A"}]`},
		{name: "未知フィールド", contentType: echo.MIMEApplicationJSON, body: `[{"id":"1234567890123456","title":"曲","artist":"A","unknown":1}]`},
		{name: "複数JSON値", contentType: echo.MIMEApplicationJSON, body: `[{"id":"1234567890123456","title":"曲","artist":"A"}] []`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := NewSongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					called = true
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			req := httptest.NewRequest(http.MethodPut, "/internal/songs", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}

			err := handler.UpdateSongs(e.NewContext(req, httptest.NewRecorder()))

			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
			assert.False(t, called)
		})
	}
}

func TestSongHandler_GetSongs_IncludeDeleted(t *testing.T) {
	// 実在の種別ID(1-4)・既定値0・他引数(bool)と紛れない値で権限の素通しを検証する。
	authenticatedAccountTypeID := 7
	tests := []struct {
		name string
		// Given
		query string
		// accountTypeID が nil のときは匿名リクエスト扱いとする。
		accountTypeID *int
		// Then
		wantCalled                 bool
		wantIncludeDeleted         bool
		wantRequesterAccountTypeID *int
		wantErrCode                string
	}{
		{name: "trueは削除済みを含めて取得する", query: "?include_deleted=true", wantCalled: true, wantIncludeDeleted: true},
		{name: "falseは削除済みを除いて取得する", query: "?include_deleted=false", accountTypeID: &authenticatedAccountTypeID, wantCalled: true, wantIncludeDeleted: false, wantRequesterAccountTypeID: &authenticatedAccountTypeID},
		{name: "不正値は既定値にせず検証エラーを返す", query: "?include_deleted=yes", wantErrCode: apierror.CodeValidationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			called := false
			var gotIncludeDeleted bool
			var gotRequesterAccountTypeID *int
			mockUsecase := &testutil.MockSongUsecase{
				GetAllSongsExcludingWorldsendFunc: func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error) {
					called = true
					gotIncludeDeleted = includeDeleted
					gotRequesterAccountTypeID = requesterAccountTypeID
					return []*entity.Song{}, nil
				},
			}
			handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/internal/songs"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.accountTypeID != nil {
				// 認証済みリクエストを再現するためコンテキストに利用者を設定する。
				c.Set("userEntity", &entity.User{AccountTypeID: *tt.accountTypeID})
			}

			// When
			err := handler.GetSongs(c)

			// Then
			assert.Equal(t, tt.wantCalled, called)
			if tt.wantErrCode != "" {
				var apiErr *apierror.APIError
				if assert.ErrorAs(t, err, &apiErr) {
					assert.Equal(t, tt.wantErrCode, apiErr.Code)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantIncludeDeleted, gotIncludeDeleted)
			// 匿名のときは nil、認証済みのときは設定した種別IDと同じ値が渡る（ポインタは指す値で比較される）。
			assert.Equal(t, tt.wantRequesterAccountTypeID, gotRequesterAccountTypeID)
		})
	}
}
