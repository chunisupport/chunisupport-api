package api_internal_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockLoginUsecase struct {
	mock.Mock
}

func (m *mockLoginUsecase) Login(ctx context.Context, idToken string, turnstileToken string, remoteIP string) (*usecase.UserOutput, error) {
	args := m.Called(ctx, idToken, turnstileToken, remoteIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*usecase.UserOutput), args.Error(1)
}

func TestLoginHandler_Login(t *testing.T) {
	e := echo.New()
	e.Validator = app.NewCustomValidator()

	t.Run("正常系: BearerトークンとTurnstileトークンでログインできる", func(t *testing.T) {
		loginUsecase := new(mockLoginUsecase)
		h := api_internal.NewLoginHandler(loginUsecase)
		loginUsecase.On("Login", mock.Anything, "firebase-id-token", "turnstile-token", "192.0.2.1").Return(&usecase.UserOutput{
			Username:    "sampleuser",
			AccountType: "PLAYER",
		}, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/internal/auth/login", bytes.NewBufferString(`{"turnstile_token":"turnstile-token"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer firebase-id-token")
		req.Header.Set(echo.HeaderXForwardedFor, "192.0.2.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "sampleuser")
		loginUsecase.AssertExpectations(t)
	})

	t.Run("異常系: Authorizationヘッダがない場合は401を返す", func(t *testing.T) {
		loginUsecase := new(mockLoginUsecase)
		h := api_internal.NewLoginHandler(loginUsecase)
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/login", bytes.NewBufferString(`{"turnstile_token":"turnstile-token"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)

		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeMissingToken, apiErr.Code)
		loginUsecase.AssertNotCalled(t, "Login", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("異常系: Turnstile検証失敗は401を返す", func(t *testing.T) {
		loginUsecase := new(mockLoginUsecase)
		h := api_internal.NewLoginHandler(loginUsecase)
		loginUsecase.On("Login", mock.Anything, "firebase-id-token", "invalid-turnstile-token", "192.0.2.1").Return(nil, usecase.ErrInvalidTurnstileToken).Once()

		req := httptest.NewRequest(http.MethodPost, "/internal/auth/login", bytes.NewBufferString(`{"turnstile_token":"invalid-turnstile-token"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer firebase-id-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)

		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidTurnstileToken, apiErr.Code)
		loginUsecase.AssertExpectations(t)
	})
}

func TestLoginHandler_Login_不正JSONは400(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "Content-Typeなし", body: `{"turnstile_token":"turnstile-token"}`},
		{name: "未知フィールド", contentType: echo.MIMEApplicationJSON, body: `{"turnstile_token":"turnstile-token","unknown":1}`},
		{name: "複数JSON値", contentType: echo.MIMEApplicationJSON, body: `{"turnstile_token":"turnstile-token"} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginUsecase := new(mockLoginUsecase)
			h := api_internal.NewLoginHandler(loginUsecase)
			e := echo.New()
			e.Validator = app.NewCustomValidator()
			req := httptest.NewRequest(http.MethodPost, "/internal/auth/login", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}
			req.Header.Set(echo.HeaderAuthorization, "Bearer firebase-id-token")

			err := h.Login(e.NewContext(req, httptest.NewRecorder()))

			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
			loginUsecase.AssertNotCalled(t, "Login", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
