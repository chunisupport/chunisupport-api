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
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSignupUsecase struct {
	mock.Mock
}

func (m *mockSignupUsecase) Signup(ctx context.Context, idToken string, username string) (*dto_internal.UserDTO, error) {
	args := m.Called(ctx, idToken, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*dto_internal.UserDTO), args.Error(1)
}

func TestSignupHandler_Signup(t *testing.T) {
	e := echo.New()
	e.Validator = app.NewCustomValidator()

	t.Run("正常系: Bearerトークンで初回登録できる", func(t *testing.T) {
		signupUsecase := new(mockSignupUsecase)
		h := api_internal.NewSignupHandler(signupUsecase)
		signupUsecase.On("Signup", mock.Anything, "firebase-id-token", "newuser").Return(&dto_internal.UserDTO{
			Username:    "newuser",
			AccountType: "PLAYER",
		}, nil).Once()

		body := `{"username":"newuser"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/signup", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer firebase-id-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Signup(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "newuser")
		signupUsecase.AssertExpectations(t)
	})

	t.Run("異常系: Authorizationヘッダがない場合は401を返す", func(t *testing.T) {
		signupUsecase := new(mockSignupUsecase)
		h := api_internal.NewSignupHandler(signupUsecase)
		body := `{"username":"newuser"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/signup", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Signup(c)

		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeMissingToken, apiErr.Code)
		signupUsecase.AssertNotCalled(t, "Signup", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("異常系: 無効トークンは401を返す", func(t *testing.T) {
		signupUsecase := new(mockSignupUsecase)
		h := api_internal.NewSignupHandler(signupUsecase)
		signupUsecase.On("Signup", mock.Anything, "invalid-token", "newuser").Return(nil, usecase.ErrInvalidIDToken).Once()

		body := `{"username":"newuser"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/signup", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Signup(c)

		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		signupUsecase.AssertExpectations(t)
	})
}
