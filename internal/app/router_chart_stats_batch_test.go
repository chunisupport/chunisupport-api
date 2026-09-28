package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type stubChartStatsBatchJobUsecase struct {
	startCalls *int
}

func (s stubChartStatsBatchJobUsecase) StartFromAdmin(context.Context, entity.ChartStatsBatchJobRequester) (*entity.ChartStatsBatchJob, error) {
	*s.startCalls++
	return nil, context.Canceled
}

func (stubChartStatsBatchJobUsecase) List(context.Context) ([]*entity.ChartStatsBatchJob, error) {
	return nil, nil
}

func (stubChartStatsBatchJobUsecase) Get(context.Context, string) (*entity.ChartStatsBatchJob, error) {
	return nil, context.Canceled
}

func TestRegisterRoutes_譜面統計バッチはADMINだけが操作できる(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		method     string
		path       string
		wantStatus int
	}{
		{name: "ADMINは履歴を取得できる", token: "admin-token", method: http.MethodGet, path: "/internal/admin/chart-stats-batch/jobs", wantStatus: http.StatusOK},
		{name: "EDITORは履歴を取得できない", token: "editor-token", method: http.MethodGet, path: "/internal/admin/chart-stats-batch/jobs", wantStatus: http.StatusForbidden},
		{name: "PLAYERは履歴を取得できない", token: "player-token", method: http.MethodGet, path: "/internal/admin/chart-stats-batch/jobs", wantStatus: http.StatusForbidden},
		{name: "EDITORは実行できない", token: "editor-token", method: http.MethodPost, path: "/internal/admin/chart-stats-batch/jobs", wantStatus: http.StatusForbidden},
		{name: "PLAYERは実行できない", token: "player-token", method: http.MethodPost, path: "/internal/admin/chart-stats-batch/jobs", wantStatus: http.StatusForbidden},
		{name: "EDITORはジョブ詳細を取得できない", token: "editor-token", method: http.MethodGet, path: "/internal/admin/chart-stats-batch/jobs/0199", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			startCalls := 0
			handlers := newAuthorizationTestHandlers()
			handlers.ChartStatsBatch = internalhandler.NewChartStatsBatchHandler(stubChartStatsBatchJobUsecase{startCalls: &startCalls})
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Zero(t, startCalls)
		})
	}
}
