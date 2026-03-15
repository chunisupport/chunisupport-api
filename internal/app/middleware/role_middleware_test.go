package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func runRequireRoleTestRequest(t *testing.T, e *echo.Echo, handler echo.HandlerFunc, user *entity.User) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if user != nil {
		c.Set("userEntity", user)
	}

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}

	return rec
}

func TestRequireRole_AdminRequired(t *testing.T) {
	tests := []struct {
		name       string
		user       *entity.User
		wantStatus int
	}{
		{name: "PLAYERは拒否", user: &entity.User{ID: 1, AccountTypeID: info.AccountTypePlayer}, wantStatus: http.StatusForbidden},
		{name: "EDITORは拒否", user: &entity.User{ID: 2, AccountTypeID: info.AccountTypeEditor}, wantStatus: http.StatusForbidden},
		{name: "ADMINは許可", user: &entity.User{ID: 3, AccountTypeID: info.AccountTypeAdmin}, wantStatus: http.StatusOK},
		{name: "未知ロールは拒否", user: &entity.User{ID: 4, AccountTypeID: 4}, wantStatus: http.StatusForbidden},
		{name: "未認証は拒否", user: nil, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := setupEchoWithErrorHandler(t)
			handler := RequireRole(info.AccountTypeAdmin)(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			rec := runRequireRoleTestRequest(t, e, handler, tt.user)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRequireRole_EditorRequired(t *testing.T) {
	tests := []struct {
		name       string
		user       *entity.User
		wantStatus int
	}{
		{name: "PLAYERは拒否", user: &entity.User{ID: 1, AccountTypeID: info.AccountTypePlayer}, wantStatus: http.StatusForbidden},
		{name: "EDITORは許可", user: &entity.User{ID: 2, AccountTypeID: info.AccountTypeEditor}, wantStatus: http.StatusOK},
		{name: "ADMINは許可", user: &entity.User{ID: 3, AccountTypeID: info.AccountTypeAdmin}, wantStatus: http.StatusOK},
		{name: "未知ロールは拒否", user: &entity.User{ID: 4, AccountTypeID: 4}, wantStatus: http.StatusForbidden},
		{name: "未認証は拒否", user: nil, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := setupEchoWithErrorHandler(t)
			handler := RequireRole(info.AccountTypeEditor)(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			rec := runRequireRoleTestRequest(t, e, handler, tt.user)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
