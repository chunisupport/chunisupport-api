package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type maintenanceStateProviderStub struct {
	enabled bool
}

func (s maintenanceStateProviderStub) Current() usecase.MaintenanceState {
	return usecase.MaintenanceState{Enabled: s.enabled}
}

type firebaseMaintenanceAuthenticatorStub struct {
	user  *entity.User
	err   error
	calls int
}

func (s *firebaseMaintenanceAuthenticatorStub) Authenticate(context.Context, string) (*entity.User, error) {
	s.calls++
	return s.user, s.err
}

func (s *firebaseMaintenanceAuthenticatorStub) AuthenticateOptional(context.Context, string) (*entity.User, error) {
	panic("AuthenticateOptionalは呼ばれない想定です")
}

type apiTokenMaintenanceValidatorStub struct {
	user  *entity.User
	token *entity.APIToken
	err   error
	calls int
}

func (s *apiTokenMaintenanceValidatorStub) Validate(context.Context, string) (*entity.User, *entity.APIToken, error) {
	s.calls++
	return s.user, s.token, s.err
}

func TestFirebaseMaintenanceMiddleware_通常時は認証せず通過する(t *testing.T) {
	// Given
	authenticator := &firebaseMaintenanceAuthenticatorStub{}
	gate := FirebaseMaintenanceMiddleware(maintenanceStateProviderStub{}, authenticator)
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := gate(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Zero(t, authenticator.calls)
}

func TestFirebaseMaintenanceMiddleware_例外経路はメンテナンス中も認証せず通過する(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "OPTIONS", method: http.MethodOptions, path: "/internal/songs"},
		{name: "ヘルスチェック", method: http.MethodGet, path: "/healthz"},
		{name: "状態確認", method: http.MethodGet, path: "/internal/system/status"},
		{name: "ログイン", method: http.MethodPost, path: "/internal/auth/login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			authenticator := &firebaseMaintenanceAuthenticatorStub{}
			gate := FirebaseMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, authenticator)
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// When
			err := gate(func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})(c)

			// Then
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Zero(t, authenticator.calls)
		})
	}
}

func TestFirebaseMaintenanceMiddleware_メンテナンス中はスタッフだけ通過する(t *testing.T) {
	tests := []struct {
		name        string
		accountType int
		wantPass    bool
	}{
		{name: "PLAYERは遮断", accountType: info.AccountTypePlayer},
		{name: "EXTDEVは遮断", accountType: info.AccountTypeExtDev},
		{name: "EDITORは通過", accountType: info.AccountTypeEditor, wantPass: true},
		{name: "ADMINは通過", accountType: info.AccountTypeAdmin, wantPass: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			user := &entity.User{ID: 1, AccountTypeID: tt.accountType}
			authenticator := &firebaseMaintenanceAuthenticatorStub{user: user}
			gate := FirebaseMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, authenticator)
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer firebase-token")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			handlerCalled := false

			// When
			err := gate(func(c *echo.Context) error {
				handlerCalled = true
				assert.Same(t, user, c.Get("userEntity"))
				return c.NoContent(http.StatusOK)
			})(c)

			// Then
			assert.Equal(t, 1, authenticator.calls)
			assert.Equal(t, tt.wantPass, handlerCalled)
			if tt.wantPass {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				return
			}
			assertMaintenanceGateError(t, err, rec)
		})
	}
}

func TestFirebaseMaintenanceMiddleware_資格情報の違いを503へマスクする(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		authenticator *firebaseMaintenanceAuthenticatorStub
	}{
		{name: "トークンなし", authenticator: &firebaseMaintenanceAuthenticatorStub{}},
		{name: "不正トークン", token: "invalid", authenticator: &firebaseMaintenanceAuthenticatorStub{err: usecase.ErrInvalidIDToken}},
		{name: "ユーザーなし", token: "missing", authenticator: &firebaseMaintenanceAuthenticatorStub{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			gate := FirebaseMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, tt.authenticator)
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
			if tt.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// When
			err := gate(func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})(c)

			// Then
			assertMaintenanceGateError(t, err, rec)
		})
	}
}

func TestFirebaseMaintenanceMiddleware_認証基盤障害を記録して503へマスクする(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})
	authenticator := &firebaseMaintenanceAuthenticatorStub{err: errors.New("firebase unavailable")}
	gate := FirebaseMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, authenticator)
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := gate(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	// Then
	assertMaintenanceGateError(t, err, rec)
	assert.Contains(t, output.String(), "Maintenance Firebase authentication failed")
	assert.Contains(t, output.String(), "firebase unavailable")
}

func TestAPITokenMaintenanceMiddleware_通常時は認証せず通過する(t *testing.T) {
	// Given
	validator := &apiTokenMaintenanceValidatorStub{}
	gate := APITokenMaintenanceMiddleware(maintenanceStateProviderStub{}, validator)
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := gate(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Zero(t, validator.calls)
}

func TestAPITokenMaintenanceMiddleware_メンテナンス中はスタッフだけ通過する(t *testing.T) {
	tests := []struct {
		name        string
		accountType int
		wantPass    bool
	}{
		{name: "PLAYERは遮断", accountType: info.AccountTypePlayer},
		{name: "EXTDEVは遮断", accountType: info.AccountTypeExtDev},
		{name: "EDITORは通過", accountType: info.AccountTypeEditor, wantPass: true},
		{name: "ADMINは通過", accountType: info.AccountTypeAdmin, wantPass: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			user := &entity.User{ID: 1, AccountTypeID: tt.accountType}
			token := &entity.APIToken{ID: 2}
			validator := &apiTokenMaintenanceValidatorStub{user: user, token: token}
			gate := APITokenMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, validator)
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/songs", nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer api-token")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			handlerCalled := false

			// When
			err := gate(func(c *echo.Context) error {
				handlerCalled = true
				assert.Same(t, user, c.Get("userEntity"))
				assert.Same(t, token, c.Get("apiToken"))
				return c.NoContent(http.StatusOK)
			})(c)

			// Then
			assert.Equal(t, 1, validator.calls)
			assert.Equal(t, tt.wantPass, handlerCalled)
			if tt.wantPass {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				return
			}
			assertMaintenanceGateError(t, err, rec)
		})
	}
}

func TestAPITokenMaintenanceMiddleware_資格情報の違いを503へマスクする(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		validator     *apiTokenMaintenanceValidatorStub
		expectedCalls int
	}{
		{name: "トークンなし", validator: new(apiTokenMaintenanceValidatorStub)},
		{
			name:          "不正トークン",
			token:         "invalid",
			validator:     &apiTokenMaintenanceValidatorStub{err: usecase.ErrInvalidAPIToken},
			expectedCalls: 1,
		},
		{
			name:          "認証情報なし",
			token:         "missing",
			validator:     new(apiTokenMaintenanceValidatorStub),
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			gate := APITokenMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, tt.validator)
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/songs", nil)
			if tt.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// When
			err := gate(func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})(c)

			// Then
			assertMaintenanceGateError(t, err, rec)
			assert.Equal(t, tt.expectedCalls, tt.validator.calls)
		})
	}
}

func TestAPITokenMaintenanceMiddleware_認証基盤障害を記録して503へマスクする(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})
	validator := &apiTokenMaintenanceValidatorStub{err: errors.New("database unavailable")}
	gate := APITokenMaintenanceMiddleware(maintenanceStateProviderStub{enabled: true}, validator)
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/songs", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := gate(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	// Then
	assertMaintenanceGateError(t, err, rec)
	assert.Contains(t, output.String(), "Maintenance API token authentication failed")
	assert.Contains(t, output.String(), "database unavailable")
}

func assertMaintenanceGateError(t *testing.T, err error, rec *httptest.ResponseRecorder) {
	t.Helper()

	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeMaintenanceMode, apiErr.Code)
	assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
}
