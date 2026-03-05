package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	dtoapi "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
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

	genreName := "POPS & ANIME"
	masterCache := &masterdata.Cache{
		Genres: map[string]masterdata.Item{
			genreName: {ID: 1, Name: genreName},
		},
	}

	testCases := []struct {
		name             string
		body             string
		expectedStatus   int
		expectedErrCode  string
		expectUsecaseHit bool
		assertUsecaseReq func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart)
	}{
		{
			name: "正常な配列で204が返る",
			body: `[{
				"id":"1234567890abcdef",
				"title":"WE曲",
				"artist":"WEアーティスト",
				"genre":"POPS & ANIME",
				"bpm":180,
				"released_at":"2024-01-01",
				"jacket":"we_jacket",
				"charts":{"WORLDSEND":{"attribute":"狂","level_star":5,"notes":2000}}
			}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart) {
				t.Helper()
				if len(songs) != 1 {
					t.Fatalf("songs len = %d, want 1", len(songs))
				}
				if len(charts) != 1 {
					t.Fatalf("charts len = %d, want 1", len(charts))
				}
				if songs[0].DisplayID != "1234567890abcdef" {
					t.Fatalf("DisplayID = %s, want 1234567890abcdef", songs[0].DisplayID)
				}
				if songs[0].GenreID == nil || *songs[0].GenreID != 1 {
					t.Fatalf("GenreID = %v, want 1", songs[0].GenreID)
				}
				if charts[0].Notes == nil || int(*charts[0].Notes) != 2000 {
					t.Fatalf("notes = %v, want 2000", charts[0].Notes)
				}
			},
		},
		{
			name: "charts省略でも楽曲情報のみ更新できる",
			body: `[{
				"id":"1234567890abcdef",
				"title":"WE曲",
				"artist":"WEアーティスト"
			}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, songs []*entity.Song, charts []*entity.WorldsendChart) {
				t.Helper()
				if len(songs) != 1 || len(charts) != 1 {
					t.Fatalf("songs/charts len = %d/%d, want 1/1", len(songs), len(charts))
				}
				if charts[0] != nil {
					t.Fatalf("chart = %v, want nil", charts[0])
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
				if len(songs) != 1 || len(charts) != 1 {
					t.Fatalf("songs/charts len = %d/%d, want 1/1", len(songs), len(charts))
				}
				if charts[0] != nil {
					t.Fatalf("chart = %v, want nil", charts[0])
				}
			},
		},
		{
			name:            "chartsキーがWORLDSEND以外でvalidation_failedが返る",
			body:            `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"MASTER":{"level_star":5}}}]`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
		{
			name:            "不正なdisplayidでvalidation_failedが返る",
			body:            `[{"id":"short","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{}}}]`,
			expectedErrCode: apierror.CodeValidationFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockUsecase := &testutil.MockWorldsendUsecase{
				GetWorldsendSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*repository.WorldsendSongWithChart, error) {
					return &repository.WorldsendSongWithChart{
						Song:  &entity.Song{ID: 100, DisplayID: displayID, OfficialIdx: "8001", IsWorldsend: true, Charts: []*entity.Chart{}},
						Chart: &entity.WorldsendChart{ID: 200, SongID: 100},
					}, nil
				},
				UpdateWorldsendSongsFunc: func(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
					called = true
					if tc.assertUsecaseReq != nil {
						tc.assertUsecaseReq(t, songs, charts)
					}
					return nil
				},
			}

			handler := NewWorldsendHandler(mockUsecase, masterCache)
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

func TestValidateAndGetWorldsendChartRequest(t *testing.T) {
	noteValue := 1234
	tests := []struct {
		name         string
		charts       map[string]*dtoapi.UpdateWorldsendChartRequest
		wantHasChart bool
		wantError    bool
	}{
		{
			name: "WORLDSENDキーのみなら成功",
			charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
				"WORLDSEND": {Notes: &noteValue},
			},
			wantHasChart: true,
			wantError:    false,
		},
		{
			name: "小文字worldsendキーは失敗",
			charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
				"worldsend": {Notes: &noteValue},
			},
			wantHasChart: false,
			wantError:    true,
		},
		{
			name: "WORLDSEND以外のキーは失敗",
			charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
				"MASTER": {Notes: &noteValue},
			},
			wantHasChart: false,
			wantError:    true,
		},
		{
			name:         "空マップは譜面更新なしとして成功",
			charts:       map[string]*dtoapi.UpdateWorldsendChartRequest{},
			wantHasChart: false,
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, hasChart, err := validateAndGetWorldsendChartRequest(tt.charts)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError %v", err, tt.wantError)
			}
			if hasChart != tt.wantHasChart {
				t.Fatalf("hasChart = %v, want %v", hasChart, tt.wantHasChart)
			}
		})
	}
}

func TestUpdateWorldsendSongs_InvalidNotes(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	invalidNotes := -1
	body := `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","charts":{"WORLDSEND":{"notes":-1}}}]`

	req := httptest.NewRequest(http.MethodPut, "/internal/songs/worldsend", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{AccountTypeID: info.AccountTypeEditor})

	mockUsecase := &testutil.MockWorldsendUsecase{
		GetWorldsendSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*repository.WorldsendSongWithChart, error) {
			n, _ := notes.NewNotes(1000)
			return &repository.WorldsendSongWithChart{
				Song:  &entity.Song{ID: 100, DisplayID: displayID, OfficialIdx: "8001", IsWorldsend: true, Charts: []*entity.Chart{}},
				Chart: &entity.WorldsendChart{ID: 200, SongID: 100, Notes: &n},
			}, nil
		},
	}

	handler := NewWorldsendHandler(mockUsecase, &masterdata.Cache{})
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
	if invalidNotes != -1 {
		t.Fatal("invalidNotes should remain unchanged")
	}
}
