package api_internal

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
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
		req := httptest.NewRequest(http.MethodPut, "/internal/worldsend-songs", bytes.NewBufferString(body))
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
		assertUsecaseReq func(t *testing.T, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters)
	}{
		{
			name:             "正常な配列で204が返る",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","genre":"POPS & ANIME","bpm":180,"released_at":"2024-01-01","jacket":"we_jacket","charts":{"WORLDSEND":{"attribute":"狂","level_star":5,"notes":2000,"notes_designer":"譜面作者A"}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) {
				t.Helper()
				if len(requests) != 1 {
					require.Failf(t, "前提条件失敗", "requests len = %d, want 1", len(requests))
				}

				req := requests[0]
				if req.DisplayID != "1234567890abcdef" {
					require.Failf(t, "前提条件失敗", "DisplayID = %s, want 1234567890abcdef", req.DisplayID)
				}
				if req.Title != "WE曲" {
					require.Failf(t, "前提条件失敗", "Title = %s, want WE曲", req.Title)
				}
				if req.Artist != "WEアーティスト" {
					require.Failf(t, "前提条件失敗", "Artist = %s, want WEアーティスト", req.Artist)
				}
				if req.Genre == nil || *req.Genre != "POPS & ANIME" {
					require.Failf(t, "前提条件失敗", "Genre = %v, want POPS & ANIME", req.Genre)
				}
				if req.BPM == nil || *req.BPM != 180 {
					require.Failf(t, "前提条件失敗", "BPM = %v, want 180", req.BPM)
				}
				if req.ReleasedAt == nil || req.ReleasedAt.Format("2006-01-02") != "2024-01-01" {
					require.Failf(t, "前提条件失敗", "ReleasedAt = %v, want 2024-01-01", req.ReleasedAt)
				}
				if req.Jacket == nil || *req.Jacket != "we_jacket" {
					require.Failf(t, "前提条件失敗", "Jacket = %v, want we_jacket", req.Jacket)
				}
				chart, ok := req.Charts["WORLDSEND"]
				if !ok || chart == nil {
					require.Failf(t, "前提条件失敗", "Charts[WORLDSEND] = %v, want non-nil", chart)
				}
				if chart.Attribute == nil || *chart.Attribute != "狂" {
					require.Failf(t, "前提条件失敗", "chart.Attribute = %v, want 狂", chart.Attribute)
				}
				if chart.LevelStar == nil || *chart.LevelStar != 5 {
					require.Failf(t, "前提条件失敗", "chart.LevelStar = %v, want 5", chart.LevelStar)
				}
				if chart.Notes == nil || *chart.Notes != 2000 {
					require.Failf(t, "前提条件失敗", "chart.Notes = %v, want 2000", chart.Notes)
				}
				if chart.NotesDesigner == nil || *chart.NotesDesigner != "譜面作者A" {
					require.Failf(t, "前提条件失敗", "chart.NotesDesigner = %v, want 譜面作者A", chart.NotesDesigner)
				}
				if masters == nil {
					require.Fail(t, "masters is nil")
				}
				if _, ok := masters.Genres["POPS & ANIME"]; !ok {
					require.Fail(t, "genre master POPS & ANIME not found")
				}

				// UTC基準で日付境界が崩れていないことを確認
				if req.ReleasedAt.Location() != time.UTC {
					require.Failf(t, "前提条件失敗", "ReleasedAt location = %v, want UTC", req.ReleasedAt.Location())
				}
			},
		},
		{
			name:             "charts省略でも楽曲情報のみ更新できる",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト"}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) {
				t.Helper()
				if len(requests) != 1 {
					require.Failf(t, "前提条件失敗", "requests len = %d, want 1", len(requests))
				}
				if requests[0].Charts != nil {
					require.Failf(t, "前提条件失敗", "charts = %v, want nil", requests[0].Charts)
				}
			},
		},
		{
			name:             "chartsがnullでも楽曲情報のみ更新できる",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":null}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) {
				t.Helper()
				if len(requests) != 1 {
					require.Failf(t, "前提条件失敗", "requests len = %d, want 1", len(requests))
				}
				if requests[0].Charts != nil {
					require.Failf(t, "前提条件失敗", "charts = %v, want nil", requests[0].Charts)
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
			expectUsecaseHit: true,
			usecaseErr:       usecase.ErrInvalidWorldsendInput,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockUsecase := &testutil.MockWorldsendUsecase{
				UpdateWorldsendSongsFunc: func(ctx context.Context, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error {
					called = true
					if tc.assertUsecaseReq != nil {
						tc.assertUsecaseReq(t, requests, masters)
					}
					return tc.usecaseErr
				},
			}

			handler := NewWorldsendHandler(mockUsecase, &masterdata.Cache{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
			c := newPutWorldsendContext(tc.body)
			rec := c.Response().Writer.(*httptest.ResponseRecorder)

			err := handler.UpdateWorldsendSongs(c)

			if tc.expectedErrCode == "" {
				if err != nil {
					require.Failf(t, "前提条件失敗", "UpdateWorldsendSongs returned error: %v", err)
				}
				if rec.Code != tc.expectedStatus {
					require.Failf(t, "前提条件失敗", "status code = %d, want %d", rec.Code, tc.expectedStatus)
				}
			} else {
				if err == nil {
					require.Fail(t, "UpdateWorldsendSongs should return error")
				}

				apiErr, ok := err.(*apierror.APIError)
				if !ok {
					require.Failf(t, "前提条件失敗", "error type = %T, want *apierror.APIError", err)
				}
				if apiErr.Code != tc.expectedErrCode {
					require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, tc.expectedErrCode)
				}
			}

			if called != tc.expectUsecaseHit {
				require.Failf(t, "前提条件失敗", "UpdateWorldsendSongs usecase called = %v, want %v", called, tc.expectUsecaseHit)
			}
		})
	}
}
