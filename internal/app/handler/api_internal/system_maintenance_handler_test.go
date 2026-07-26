package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type systemMaintenanceUsecaseStub struct {
	state       usecase.MaintenanceState
	updateErr   error
	updateCalls int
	actorUserID int
	enabled     bool
	comment     string
}

func (s *systemMaintenanceUsecaseStub) Current() usecase.MaintenanceState {
	return s.state
}

func (s *systemMaintenanceUsecaseStub) Update(
	_ context.Context,
	actorUserID int,
	enabled bool,
	comment string,
) (usecase.MaintenanceState, error) {
	s.updateCalls++
	s.actorUserID = actorUserID
	s.enabled = enabled
	s.comment = comment
	return s.state, s.updateErr
}

func TestSystemMaintenanceHandler_Status(t *testing.T) {
	updatedAt := time.Date(2026, 7, 26, 3, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		state      usecase.MaintenanceState
		wantStatus string
	}{
		{
			name:       "通常時",
			state:      usecase.MaintenanceState{UpdatedAt: updatedAt},
			wantStatus: "operational",
		},
		{
			name:       "メンテナンス中",
			state:      usecase.MaintenanceState{Enabled: true, Comment: "データ更新中です", UpdatedAt: updatedAt},
			wantStatus: "maintenance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			handler := NewSystemMaintenanceHandler(&systemMaintenanceUsecaseStub{state: tt.state})
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/system/status", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// When
			err := handler.Status(c)

			// Then
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))

			var response systemMaintenanceResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.wantStatus, response.Status)
			assert.Equal(t, tt.state.Comment, response.Comment)
			assert.Equal(t, updatedAt, response.UpdatedAt)
			assert.NotContains(t, rec.Body.String(), "updated_by_user_id")
		})
	}
}

func TestSystemMaintenanceHandler_Update_ADMINの操作をユースケースへ渡す(t *testing.T) {
	// Given
	state := usecase.MaintenanceState{
		Enabled:   true,
		Comment:   "データ更新中です",
		UpdatedAt: time.Date(2026, 7, 26, 3, 30, 0, 0, time.UTC),
	}
	maintenanceUsecase := &systemMaintenanceUsecaseStub{state: state}
	handler := NewSystemMaintenanceHandler(maintenanceUsecase)
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/internal/admin/maintenance",
		bytes.NewBufferString(`{"enabled":true,"comment":"データ更新中です"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 42})

	// When
	err := handler.Update(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.Equal(t, 1, maintenanceUsecase.updateCalls)
	assert.Equal(t, 42, maintenanceUsecase.actorUserID)
	assert.True(t, maintenanceUsecase.enabled)
	assert.Equal(t, "データ更新中です", maintenanceUsecase.comment)

	var response systemMaintenanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "maintenance", response.Status)
	assert.Equal(t, state.Comment, response.Comment)
	assert.Equal(t, state.UpdatedAt, response.UpdatedAt)
}

func TestSystemMaintenanceHandler_Update_enabled未指定は400(t *testing.T) {
	// Given
	maintenanceUsecase := new(systemMaintenanceUsecaseStub)
	handler := NewSystemMaintenanceHandler(maintenanceUsecase)
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/internal/admin/maintenance",
		bytes.NewBufferString(`{"comment":"データ更新中です"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 42})

	// When
	err := handler.Update(c)

	// Then
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Zero(t, maintenanceUsecase.updateCalls)
}

func TestSystemMaintenanceHandler_Update_不正JSONは400(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "Content-Typeなし", body: `{"enabled":true,"comment":"更新中"}`},
		{name: "未知フィールド", contentType: echo.MIMEApplicationJSON, body: `{"enabled":true,"comment":"更新中","unknown":1}`},
		{name: "複数JSON値", contentType: echo.MIMEApplicationJSON, body: `{"enabled":true,"comment":"更新中"} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			maintenanceUsecase := new(systemMaintenanceUsecaseStub)
			handler := NewSystemMaintenanceHandler(maintenanceUsecase)
			e := echo.New()
			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodPut,
				"/internal/admin/maintenance",
				bytes.NewBufferString(tt.body),
			)
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("userEntity", &entity.User{ID: 42})

			// When
			err := handler.Update(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.Zero(t, maintenanceUsecase.updateCalls)
		})
	}
}

func TestSystemMaintenanceHandler_Update_不正コメントを400へ変換する(t *testing.T) {
	// Given
	maintenanceUsecase := &systemMaintenanceUsecaseStub{updateErr: usecase.ErrInvalidMaintenanceComment}
	handler := NewSystemMaintenanceHandler(maintenanceUsecase)
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/internal/admin/maintenance",
		bytes.NewBufferString(`{"enabled":true,"comment":"更新中"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 42})

	// When
	err := handler.Update(c)

	// Then
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.ErrorIs(t, apiErr.Internal, usecase.ErrInvalidMaintenanceComment)
}

func TestSystemMaintenanceHandler_Update_認証済みユーザーがなければ401(t *testing.T) {
	// Given
	handler := NewSystemMaintenanceHandler(new(systemMaintenanceUsecaseStub))
	e := echo.New()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/internal/admin/maintenance",
		bytes.NewBufferString(`{"enabled":false,"comment":""}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handler.Update(c)

	// Then
	assert.ErrorIs(t, err, apierror.ErrUnauthorized)
}

var _ systemMaintenanceUsecase = (*systemMaintenanceUsecaseStub)(nil)
