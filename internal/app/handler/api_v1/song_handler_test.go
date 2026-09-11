package api_v1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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
	return tv.validator.Struct(i)
}

func TestV1SongHandler_UpdateSongs(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newContext := func(body string) *echo.Context {
		req := httptest.NewRequest(http.MethodPut, "/v1/songs", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	tests := []struct {
		name             string
		body             string
		wantStatus       int
		wantErrCode      string
		wantUsecaseCall  bool
		assertUsecaseReq func(t *testing.T, requests []*usecase.UpdateSongInput)
	}{
		{
			name:            "正常な配列で204を返す",
			body:            `[{"id":"1234567890abcdef","title":"テスト楽曲","artist":"テストアーティスト","charts":{"MASTER":{"const":14.5,"is_const_unknown":false,"notes":1234,"notes_designer":"譜面作者A"}}}]`,
			wantStatus:      http.StatusNoContent,
			wantUsecaseCall: true,
			assertUsecaseReq: func(t *testing.T, requests []*usecase.UpdateSongInput) {
				t.Helper()
				require.Len(t, requests, 1)
				assert.Equal(t, "1234567890abcdef", requests[0].DisplayID)
				require.Contains(t, requests[0].Charts, "MASTER")
				assert.InDelta(t, 14.5, requests[0].Charts["MASTER"].Const, 0.0001)
			},
		},
		{
			name:        "トップレベルnullはvalidation_failedを返す",
			body:        `null`,
			wantErrCode: apierror.CodeValidationFailed,
		},
		{
			name:        "不正なdisplay_idはvalidation_failedを返す",
			body:        `[{"id":"short","title":"テスト楽曲","artist":"テストアーティスト"}]`,
			wantErrCode: apierror.CodeValidationFailed,
		},
		{
			name:        "chartsのnull要素はvalidation_failedを返す",
			body:        `[{"id":"1234567890abcdef","title":"テスト楽曲","artist":"テストアーティスト","charts":{"MASTER":null}}]`,
			wantErrCode: apierror.CodeValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecaseCalled := false
			handler := NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					usecaseCalled = true
					if tt.assertUsecaseReq != nil {
						tt.assertUsecaseReq(t, requests)
					}
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})

			c := newContext(tt.body)
			response, _ := echo.UnwrapResponse(c.Response())
			rec := response.ResponseWriter.(*httptest.ResponseRecorder)

			err := handler.UpdateSongs(c)

			if tt.wantErrCode == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			} else {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantErrCode, apiErr.Code)
			}
			assert.Equal(t, tt.wantUsecaseCall, usecaseCalled)
		})
	}
}

func TestV1SongHandler_UpdateChartConstant(t *testing.T) {
	// Given
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	var got usecase.UpdateChartConstantInput
	updatedSong := &entity.Song{
		DisplayID:   "display-id",
		OfficialIdx: "123",
		Title:       "更新対象楽曲",
		Artist:      "アーティスト",
		Charts:      []*entity.Chart{},
	}
	h := NewV1SongHandler(&testutil.MockSongUsecase{
		UpdateChartConstantFunc: func(_ context.Context, input usecase.UpdateChartConstantInput) (*entity.Song, error) {
			got = input
			return updatedSong, nil
		},
	}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/songs/chart-constant", bytes.NewBufferString(
		`{"official_idx":"123","difficulty":"MAS","const":14.7}`,
	))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := h.UpdateChartConstant(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{
		"id":"display-id",
		"title":"更新対象楽曲",
		"reading":null,
		"artist":"アーティスト",
		"genre":null,
		"bpm":null,
		"release":null,
		"jacket":null,
		"official_idx":"123",
		"maxop":90,
		"is_maxop_unknown":false,
		"op_target_difficulty":null,
		"is_new":false,
		"charts":{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null}
	}`, rec.Body.String())
	assert.Equal(t, usecase.UpdateChartConstantInput{
		OfficialIdx: "123",
		Difficulty:  "MAS",
		Const:       14.7,
	}, got)
}

func TestV1SongHandler_UpdateChartConstant_const未指定を拒否する(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "constがない",
			body: `{"official_idx":"123","difficulty":"MAS"}`,
		},
		{
			name: "constがnull",
			body: `{"official_idx":"123","difficulty":"MAS","const":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			called := false
			h := NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateChartConstantFunc: func(_ context.Context, _ usecase.UpdateChartConstantInput) (*entity.Song, error) {
					called = true
					return nil, nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			req := httptest.NewRequest(http.MethodPatch, "/v1/songs/chart-constant", bytes.NewBufferString(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := e.NewContext(req, httptest.NewRecorder())

			// When
			err := h.UpdateChartConstant(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
			assert.False(t, called)
		})
	}
}

func TestV1SongHandler_GetSongRejectsInvalidDisplayID(t *testing.T) {
	e := echo.New()
	called := false
	handler := NewV1SongHandler(&testutil.MockSongUsecase{
		GetSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error) {
			called = true
			return nil, nil
		},
	}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})

	req := httptest.NewRequest(http.MethodGet, "/v1/songs/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})

	err := handler.GetSong(c)

	var apiErr *apierror.APIError
	if assert.ErrorAs(t, err, &apiErr) {
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	}
	assert.False(t, called)
}

func TestV1SongHandler_GetSongはidパスパラメータを使用する(t *testing.T) {
	// Given
	const id = "0123456789abcdef"
	var actualID string
	handler := NewV1SongHandler(&testutil.MockSongUsecase{
		GetSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error) {
			actualID = displayID
			return &entity.Song{DisplayID: displayID}, nil
		},
	}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/v1/songs/"+id, nil), httptest.NewRecorder())
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

	// When
	err := handler.GetSong(c)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, id, actualID)
}

func TestV1SongHandler_UpdateSongs_不正JSONは400(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "Content-Typeなし", body: `[{"id":"1234567890abcdef","title":"曲","artist":"A"}]`},
		{name: "未知フィールド", contentType: echo.MIMEApplicationJSON, body: `[{"id":"1234567890abcdef","title":"曲","artist":"A","unknown":1}]`},
		{name: "複数JSON値", contentType: echo.MIMEApplicationJSON, body: `[{"id":"1234567890abcdef","title":"曲","artist":"A"}] []`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					called = true
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			req := httptest.NewRequest(http.MethodPut, "/v1/songs", bytes.NewBufferString(tt.body))
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

func stringPtr(value string) *string {
	return &value
}
