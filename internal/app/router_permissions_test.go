package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type permissionMasterStub struct{ usecase.MasterDataUsecase }

func (permissionMasterStub) GetPermissions(context.Context) []string {
	return []string{"PLAYER", "EDITOR", "ADMIN", "EXTDEV"}
}

type permissionAuthenticator struct{ stubFirebaseAuthenticator }

func (permissionAuthenticator) Authenticate(ctx context.Context, token string) (*entity.User, error) {
	if token == "extdev-token" {
		return &entity.User{ID: 4, AccountTypeID: info.AccountTypeExtDev}, nil
	}
	return authenticateTestUser(token), nil
}

func TestRegisterRoutes_権限一覧はADMIN専用(t *testing.T) {
	tests := []struct {
		name, token string
		status      int
	}{
		{"ADMIN", "admin-token", http.StatusOK},
		{"EDITOR", "editor-token", http.StatusForbidden},
		{"EXTDEV", "extdev-token", http.StatusForbidden},
		{"PLAYER", "player-token", http.StatusForbidden},
		{"未認証", "", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := newAuthorizationTestHandlers()
			handlers.MasterData = internalhandler.NewMasterDataHandler(permissionMasterStub{})
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			auth := permissionAuthenticator{}
			registerRoutes(e, handlers, auth, auth, nil, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequest(http.MethodGet, "/internal/master/permissions", nil)
			if tt.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, tt.status, rec.Code)
			if tt.status == http.StatusOK {
				assert.JSONEq(t, `{"permissions":["PLAYER","EDITOR","ADMIN","EXTDEV"]}`, rec.Body.String())
			}
		})
	}
}

type permissionChangeStub struct{ called bool }

func (s *permissionChangeStub) ChangePermission(context.Context, *entity.User, string, string) error {
	s.called = true
	return nil
}

func TestRegisterRoutes_権限変更はADMIN専用(t *testing.T) {
	tests := []struct {
		name, token string
		status      int
	}{
		{"ADMIN", "admin-token", http.StatusNoContent},
		{"EDITOR", "editor-token", http.StatusForbidden},
		{"EXTDEV", "extdev-token", http.StatusForbidden},
		{"PLAYER", "player-token", http.StatusForbidden},
		{"未認証", "", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &permissionChangeStub{}
			handlers := newAuthorizationTestHandlers()
			handlers.UserPermission = internalhandler.NewUserPermissionHandler(uc)
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			auth := permissionAuthenticator{}
			registerRoutes(e, handlers, auth, auth, nil, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequest(http.MethodPatch, "/internal/users/target/permission", strings.NewReader(`{"permission":"EDITOR"}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if tt.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, tt.status == http.StatusNoContent, uc.called)
		})
	}
}
