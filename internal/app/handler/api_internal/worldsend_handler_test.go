package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func TestUpdateWorldsendSongs(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newPutWorldsendContext := func(body string) echo.Context {
		req := httptest.NewRequest(http.MethodPut, "/internal/songs/worldsend", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{AccountTypeID: info.AccountTypeEditor})
		return c
	}

	testCases := []struct {
		name             string
		body             string
		expectedStatus   int
		expectedErrCode  string
		expectUsecaseHit bool
		usecaseErr       error
		assertUsecaseReq func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart)
	}{
		{
			name:             "正常な配列で204が返る",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","genre":"POPS & ANIME","bpm":180,"released_at":"2024-01-01","jacket":"we_jacket","charts":{"WORLDSEND":{"attribute":"狂","level_star":5,"notes":2000}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart) {
				t.Helper()
				if len(songs) != 1 {
					t.Fatalf("songs len = %d, want 1", len(songs))
				}
				if songs[0].DisplayID != "1234567890abcdef" {
					t.Fatalf("DisplayID = %s, want 1234567890abcdef", songs[0].DisplayID)
				}
				if songs[0].GenreID == nil || *songs[0].GenreID != 1 {
					t.Fatalf("genreID = %v, want 1", songs[0].GenreID)
				}
				if len(charts) != 1 || charts[0] == nil {
					t.Fatalf("charts = %v, want 1 chart", charts)
				}
				if charts[0].Notes == nil || int(*charts[0].Notes) != 2000 {
					t.Fatalf("notes = %v, want 2000", charts[0].Notes)
				}
				if charts[0].LevelStar == nil || *charts[0].LevelStar != levelstar.LevelStar(5) {
					t.Fatalf("levelStar = %v, want 5", charts[0].LevelStar)
				}
				if charts[0].Attribute == nil || *charts[0].Attribute != "狂" {
					t.Fatalf("attribute = %v, want 狂", charts[0].Attribute)
				}
			},
		},
		{
			name:             "charts省略でも楽曲情報のみ更新できる",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト"}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart) {
				t.Helper()
				if len(songs) != 1 {
					t.Fatalf("songs len = %d, want 1", len(songs))
				}
				if len(charts) != 1 || charts[0] != nil {
					t.Fatalf("charts = %v, want [nil]", charts)
				}
			},
		},
		{
			name:             "chartsがnullでも楽曲情報のみ更新できる",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":null}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart) {
				t.Helper()
				if len(songs) != 1 {
					t.Fatalf("songs len = %d, want 1", len(songs))
				}
				if len(charts) != 1 || charts[0] != nil {
					t.Fatalf("charts = %v, want [nil]", charts)
				}
			},
		},
		{
			name:            "不正なdisplayidでvalidation_failedが返る",
			body:            `[{"id":"short","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{}}}]`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
		{
			name:             "usecaseで入力エラーならvalidation_failedが返る",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"MASTER":{"level_star":5}}}]`,
			expectedErrCode:  apierror.CodeValidationFailed,
			expectUsecaseHit: false,
			usecaseErr:       usecase.ErrInvalidWorldsendInput,
		},
		{
			name:             "chartsキーが複数ならvalidation_failedが返る",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{},"MASTER":{}}}]`,
			expectedErrCode:  apierror.CodeValidationFailed,
			expectUsecaseHit: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockUsecase := &testutil.MockWorldsendUsecase{
				UpdateWorldsendSongsFunc: func(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
					called = true
					if tc.assertUsecaseReq != nil {
						tc.assertUsecaseReq(t, songs, charts)
					}
					return tc.usecaseErr
				},
			}

			handler := NewWorldsendHandler(mockUsecase, &masterdata.Cache{Genres: map[string]masterdata.Item{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
			c := newPutWorldsendContext(tc.body)
			rec := c.Response().Writer.(*httptest.ResponseRecorder)

			err := handler.UpdateWorldsendSongs(c)

			if tc.expectedErrCode == "" {
				if err != nil {
					t.Fatalf("UpdateWorldsendSongs returned error: %v", err)
				}
				if rec.Code != tc.expectedStatus {
					t.Fatalf("status code = %d, want %d", rec.Code, tc.expectedStatus)
				}
			} else {
				if err == nil {
					t.Fatal("UpdateWorldsendSongs should return error")
				}

				apiErr, ok := err.(*apierror.APIError)
				if !ok {
					t.Fatalf("error type = %T, want *apierror.APIError", err)
				}
				if apiErr.Code != tc.expectedErrCode {
					t.Fatalf("api error code = %s, want %s", apiErr.Code, tc.expectedErrCode)
				}
			}

			if called != tc.expectUsecaseHit {
				t.Fatalf("UpdateWorldsendSongs usecase called = %v, want %v", called, tc.expectUsecaseHit)
			}
		})
	}
}

func TestUpdateWorldsendSongs_不正なlevel_starはvalidation_failedを返す(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	req := httptest.NewRequest(http.MethodPut, "/internal/songs/worldsend", bytes.NewBufferString(`[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{"level_star":0}}}]`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{AccountTypeID: info.AccountTypeEditor})

	called := false
	handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
		UpdateWorldsendSongsFunc: func(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
			called = true
			return nil
		},
	}, &masterdata.Cache{})

	err := handler.UpdateWorldsendSongs(c)
	if err == nil {
		t.Fatal("UpdateWorldsendSongs should return error")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("error type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeValidationFailed {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeValidationFailed)
	}
	if called {
		t.Fatal("usecase should not be called")
	}
}

func TestUpdateWorldsendSongs_不正なnotesはvalidation_failedを返す(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	req := httptest.NewRequest(http.MethodPut, "/internal/songs/worldsend", bytes.NewBufferString(`[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{"notes":-1}}}]`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{AccountTypeID: info.AccountTypeEditor})

	called := false
	handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
		UpdateWorldsendSongsFunc: func(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
			called = true
			return nil
		},
	}, &masterdata.Cache{})

	err := handler.UpdateWorldsendSongs(c)
	if err == nil {
		t.Fatal("UpdateWorldsendSongs should return error")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("error type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeValidationFailed {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeValidationFailed)
	}
	if called {
		t.Fatal("usecase should not be called")
	}
}

func TestUpdateWorldsendSongs_不正なgenreはvalidation_failedを返す(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	req := httptest.NewRequest(http.MethodPut, "/internal/songs/worldsend", bytes.NewBufferString(`[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","genre":"UNKNOWN"}]`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{AccountTypeID: info.AccountTypeEditor})

	called := false
	handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
		UpdateWorldsendSongsFunc: func(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
			called = true
			return nil
		},
	}, &masterdata.Cache{Genres: map[string]masterdata.Item{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})

	err := handler.UpdateWorldsendSongs(c)
	if err == nil {
		t.Fatal("UpdateWorldsendSongs should return error")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("error type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeValidationFailed {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeValidationFailed)
	}
	if called {
		t.Fatal("usecase should not be called")
	}
}

func TestUpdateWorldsendSongs_Charts変換でnotes型が保持される(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	req := httptest.NewRequest(http.MethodPut, "/internal/songs/worldsend", bytes.NewBufferString(`[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{"notes":2000}}}]`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{AccountTypeID: info.AccountTypeEditor})

	handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
		UpdateWorldsendSongsFunc: func(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
			if len(charts) != 1 || charts[0] == nil || charts[0].Notes == nil {
				t.Fatalf("charts notes が変換されていません: %+v", charts)
			}
			if *charts[0].Notes != notes.Notes(2000) {
				t.Fatalf("notes = %v, want 2000", *charts[0].Notes)
			}
			return nil
		},
	}, &masterdata.Cache{})

	err := handler.UpdateWorldsendSongs(c)
	if err != nil {
		t.Fatalf("UpdateWorldsendSongs returned error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
