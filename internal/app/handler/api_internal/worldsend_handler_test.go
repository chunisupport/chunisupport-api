package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
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
		assertUsecaseReq func(t *testing.T, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters)
	}{
		{
			name:             "正常な配列で204が返る",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","genre":"POPS & ANIME","bpm":180,"released_at":"2024-01-01","jacket":"we_jacket","charts":{"WORLDSEND":{"attribute":"狂","level_star":5,"notes":2000}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) {
				t.Helper()
				if len(requests) != 1 {
					t.Fatalf("requests len = %d, want 1", len(requests))
				}

				req := requests[0]
				if req.DisplayID != "1234567890abcdef" {
					t.Fatalf("DisplayID = %s, want 1234567890abcdef", req.DisplayID)
				}
				if req.Title != "WE曲" {
					t.Fatalf("Title = %s, want WE曲", req.Title)
				}
				if req.Artist != "WEアーティスト" {
					t.Fatalf("Artist = %s, want WEアーティスト", req.Artist)
				}
				if req.Genre == nil || *req.Genre != "POPS & ANIME" {
					t.Fatalf("Genre = %v, want POPS & ANIME", req.Genre)
				}
				if req.BPM == nil || *req.BPM != 180 {
					t.Fatalf("BPM = %v, want 180", req.BPM)
				}
				if req.ReleasedAt == nil || req.ReleasedAt.Format("2006-01-02") != "2024-01-01" {
					t.Fatalf("ReleasedAt = %v, want 2024-01-01", req.ReleasedAt)
				}
				if req.Jacket == nil || *req.Jacket != "we_jacket" {
					t.Fatalf("Jacket = %v, want we_jacket", req.Jacket)
				}
				chart, ok := req.Charts["WORLDSEND"]
				if !ok || chart == nil {
					t.Fatalf("Charts[WORLDSEND] = %v, want non-nil", chart)
				}
				if chart.Attribute == nil || *chart.Attribute != "狂" {
					t.Fatalf("chart.Attribute = %v, want 狂", chart.Attribute)
				}
				if chart.LevelStar == nil || *chart.LevelStar != 5 {
					t.Fatalf("chart.LevelStar = %v, want 5", chart.LevelStar)
				}
				if chart.Notes == nil || *chart.Notes != 2000 {
					t.Fatalf("chart.Notes = %v, want 2000", chart.Notes)
				}
				if masters == nil {
					t.Fatal("masters is nil")
				}
				if _, ok := masters.Genres["POPS & ANIME"]; !ok {
					t.Fatal("genre master POPS & ANIME not found")
				}

				// UTC基準で日付境界が崩れていないことを確認
				if req.ReleasedAt.Location() != time.UTC {
					t.Fatalf("ReleasedAt location = %v, want UTC", req.ReleasedAt.Location())
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
					t.Fatalf("requests len = %d, want 1", len(requests))
				}
				if requests[0].Charts != nil {
					t.Fatalf("charts = %v, want nil", requests[0].Charts)
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
					t.Fatalf("requests len = %d, want 1", len(requests))
				}
				if requests[0].Charts != nil {
					t.Fatalf("charts = %v, want nil", requests[0].Charts)
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
