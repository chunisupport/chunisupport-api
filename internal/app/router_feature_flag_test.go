package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes_FeatureFlagProbe(t *testing.T) {
	tests := []struct {
		name string
		// Given
		flags featureflag.Flags
		token string
		// Then
		status int
		body   string
	}{
		{name: "有効な環境ではADMINが取得できる", flags: featureflag.New(featureflag.FeatureFlagProbe), token: "admin-token", status: http.StatusOK, body: `{"enabled":true}`},
		{name: "有効な環境でもADMIN以外は取得できない", flags: featureflag.New(featureflag.FeatureFlagProbe), token: "editor-token", status: http.StatusForbidden},
		{name: "無効な環境ではルートがなく404になる", flags: featureflag.New(), token: "admin-token", status: http.StatusNotFound},
		{name: "有効な環境では未認証だと401になる", flags: featureflag.New(featureflag.FeatureFlagProbe), token: "", status: http.StatusUnauthorized},
		{name: "無効な環境では認証より先に404になる", flags: featureflag.New(), token: "", status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			auth := permissionAuthenticator{}
			registerRoutes(e, newAuthorizationTestHandlers(), auth, auth, nil, stubMaintenanceUsecase{}, config.Config{FeatureFlags: tt.flags})
			req := httptest.NewRequest(http.MethodGet, "/internal/admin/feature-flag-probe", nil)
			if tt.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.status, rec.Code)
			if tt.body != "" {
				assert.JSONEq(t, tt.body, rec.Body.String())
			}
		})
	}
}
