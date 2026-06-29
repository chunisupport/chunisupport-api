package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
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
		e.HTTPErrorHandler(c, err)
	}

	return rec
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		requiredRoleID int
		user           *entity.User
		wantStatus     int
	}{
		{name: "AdminRequired/PLAYERは拒否", requiredRoleID: info.AccountTypeAdmin, user: &entity.User{ID: 1, AccountTypeID: info.AccountTypePlayer}, wantStatus: http.StatusForbidden},
		{name: "AdminRequired/EDITORは拒否", requiredRoleID: info.AccountTypeAdmin, user: &entity.User{ID: 2, AccountTypeID: info.AccountTypeEditor}, wantStatus: http.StatusForbidden},
		{name: "AdminRequired/ADMINは許可", requiredRoleID: info.AccountTypeAdmin, user: &entity.User{ID: 3, AccountTypeID: info.AccountTypeAdmin}, wantStatus: http.StatusOK},
		{name: "AdminRequired/未知ロールは拒否", requiredRoleID: info.AccountTypeAdmin, user: &entity.User{ID: 4, AccountTypeID: 4}, wantStatus: http.StatusForbidden},
		{name: "AdminRequired/未認証は拒否", requiredRoleID: info.AccountTypeAdmin, user: nil, wantStatus: http.StatusUnauthorized},
		{name: "EditorRequired/PLAYERは拒否", requiredRoleID: info.AccountTypeEditor, user: &entity.User{ID: 1, AccountTypeID: info.AccountTypePlayer}, wantStatus: http.StatusForbidden},
		{name: "EditorRequired/EDITORは許可", requiredRoleID: info.AccountTypeEditor, user: &entity.User{ID: 2, AccountTypeID: info.AccountTypeEditor}, wantStatus: http.StatusOK},
		{name: "EditorRequired/ADMINは許可", requiredRoleID: info.AccountTypeEditor, user: &entity.User{ID: 3, AccountTypeID: info.AccountTypeAdmin}, wantStatus: http.StatusOK},
		{name: "EditorRequired/未知ロールは拒否", requiredRoleID: info.AccountTypeEditor, user: &entity.User{ID: 4, AccountTypeID: 4}, wantStatus: http.StatusForbidden},
		{name: "EditorRequired/未認証は拒否", requiredRoleID: info.AccountTypeEditor, user: nil, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := setupEchoWithErrorHandler(t)
			handler := RequireRole(tt.requiredRoleID)(func(c *echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			// When
			rec := runRequireRoleTestRequest(t, e, handler, tt.user)

			// Then
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
