package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dtoapi "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
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
		assertUsecaseReq func(t *testing.T, requests []*dtoapi.UpdateWorldsendSongRequest)
	}{
		{
			name:             "正常な配列で204が返る",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト","genre":"POPS & ANIME","bpm":180,"released_at":"2024-01-01","jacket":"we_jacket","charts":{"WORLDSEND":{"attribute":"狂","level_star":5,"notes":2000}}}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*dtoapi.UpdateWorldsendSongRequest) {
				t.Helper()
				if len(requests) != 1 {
					t.Fatalf("requests len = %d, want 1", len(requests))
				}
				if requests[0].DisplayID != "1234567890abcdef" {
					t.Fatalf("DisplayID = %s, want 1234567890abcdef", requests[0].DisplayID)
				}
				if requests[0].Genre == nil || *requests[0].Genre != "POPS & ANIME" {
					t.Fatalf("genre = %v, want POPS & ANIME", requests[0].Genre)
				}
				if requests[0].Charts["WORLDSEND"].Notes == nil || *requests[0].Charts["WORLDSEND"].Notes != 2000 {
					t.Fatalf("notes = %v, want 2000", requests[0].Charts["WORLDSEND"].Notes)
				}
			},
		},
		{
			name:             "charts省略でも楽曲情報のみ更新できる",
			body:             `[{"id":"1234567890abcdef","title":"WE曲","artist":"WEアーティスト"}]`,
			expectedStatus:   http.StatusNoContent,
			expectUsecaseHit: true,
			assertUsecaseReq: func(t *testing.T, requests []*dtoapi.UpdateWorldsendSongRequest) {
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
			assertUsecaseReq: func(t *testing.T, requests []*dtoapi.UpdateWorldsendSongRequest) {
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
				UpdateWorldsendSongsFunc: func(ctx context.Context, requests []*dtoapi.UpdateWorldsendSongRequest) error {
					called = true
					if tc.assertUsecaseReq != nil {
						tc.assertUsecaseReq(t, requests)
					}
					return tc.usecaseErr
				},
			}

			handler := NewWorldsendHandler(mockUsecase, &masterdata.Cache{})
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
