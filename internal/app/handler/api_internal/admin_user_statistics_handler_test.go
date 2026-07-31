package api_internal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminUserStatisticsUsecaseStub struct {
	result usecase.AdminUserStatisticsOutput
	err    error
}

func (s adminUserStatisticsUsecaseStub) Get(context.Context) (usecase.AdminUserStatisticsOutput, error) {
	return s.result, s.err
}

func TestAdminUserStatisticsHandler_Get(t *testing.T) {
	// Given
	handler := NewAdminUserStatisticsHandler(adminUserStatisticsUsecaseStub{result: usecase.AdminUserStatisticsOutput{
		TotalUsers:                 100,
		UsersWithPlayerData:        80,
		ActivePlayerDataLast30Days: 50,
	}})
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/admin/user-stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handler.Get(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, map[string]int{
		"total_users":                     100,
		"users_with_player_data":          80,
		"active_player_data_last_30_days": 50,
	}, response)
}

func TestAdminUserStatisticsHandler_Get_取得失敗は内部エラー(t *testing.T) {
	// Given
	handler := NewAdminUserStatisticsHandler(adminUserStatisticsUsecaseStub{err: errors.New("query failed")})
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/admin/user-stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handler.Get(c)

	// Then
	assert.ErrorIs(t, err, apierror.ErrInternalError)
}
