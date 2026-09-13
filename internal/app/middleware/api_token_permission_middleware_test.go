package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/apitokenpermission"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAPITokenWrite(t *testing.T) {
	tests := []struct {
		name       string
		permission apitokenpermission.APITokenPermission
		setToken   bool
		wantStatus int
	}{
		{name: "readは拒否", permission: apitokenpermission.Read, setToken: true, wantStatus: http.StatusForbidden},
		{name: "read_writeは通過", permission: apitokenpermission.ReadWrite, setToken: true, wantStatus: http.StatusOK},
		{name: "トークンなしは未認証", setToken: false, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, "/v1/songs", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.setToken {
				c.Set(contextKeyAPIToken, &entity.APIToken{Permission: tt.permission})
			}

			handlerCalled := false
			err := RequireAPITokenWrite()(func(c *echo.Context) error {
				handlerCalled = true
				return c.NoContent(http.StatusOK)
			})(c)

			if tt.wantStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			}
			assert.Equal(t, tt.wantStatus == http.StatusOK, handlerCalled)
		})
	}
}
