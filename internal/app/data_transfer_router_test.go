package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutesDataTransferOperationsAreAlwaysRegisteredAndAuthenticated(t *testing.T) {
	paths := []string{
		"/internal/me/data-transfer/export",
		"/internal/me/data-transfer/validate",
		"/internal/me/data-transfer/import",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			handlers := newAuthorizationTestHandlers()
			handlers.DataTransfer = internalhandler.NewDataTransferHandler()
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})

			unauthenticated := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewBufferString("{}"))
			unauthenticatedRecorder := httptest.NewRecorder()
			e.ServeHTTP(unauthenticatedRecorder, unauthenticated)
			assert.Equal(t, http.StatusUnauthorized, unauthenticatedRecorder.Code)

			authenticated := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewBufferString("{}"))
			authenticated.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
			authenticatedRecorder := httptest.NewRecorder()
			e.ServeHTTP(authenticatedRecorder, authenticated)
			assert.NotEqual(t, http.StatusNotFound, authenticatedRecorder.Code)
		})
	}
}
func TestRegisterRoutesDataTransferCapabilitiesIsNotRegistered(t *testing.T) {
	handlers := newAuthorizationTestHandlers()
	handlers.DataTransfer = internalhandler.NewDataTransferHandler()
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/internal/me/data-transfer/capabilities", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
func TestRequestBodyLimitMiddlewareUsesTransferSpecificLimit(t *testing.T) {
	body := bytes.Repeat([]byte("a"), info.RequestBodyLimit+1)
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "通常APIは5MiBを超えると拒否する", path: "/internal/other", wantStatus: http.StatusRequestEntityTooLarge},
		{name: "validateは移行用上限まで受け付ける", path: "/internal/me/data-transfer/validate", wantStatus: http.StatusNoContent},
		{name: "importは移行用上限まで受け付ける", path: "/internal/me/data-transfer/import", wantStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(newRequestBodyLimitMiddleware())
			e.POST(tt.path, func(context *echo.Context) error {
				_, err := context.Request().Body.Read(make([]byte, len(body)))
				if err != nil {
					return err
				}
				return context.NoContent(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(body))
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRegisterRoutesDataTransferOperationsShareRateLimit(t *testing.T) {
	handlers := newAuthorizationTestHandlers()
	handlers.DataTransfer = internalhandler.NewDataTransferHandler()
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
	paths := []string{
		"/internal/me/data-transfer/export",
		"/internal/me/data-transfer/validate",
		"/internal/me/data-transfer/import",
		"/internal/me/data-transfer/export",
		"/internal/me/data-transfer/validate",
		"/internal/me/data-transfer/import",
	}
	for index, path := range paths {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString("{}"))
		req.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if index < info.DataTransferRateLimitRequests {
			assert.NotEqual(t, http.StatusTooManyRequests, rec.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		}
	}
}
