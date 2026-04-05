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
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockFirebaseLinkUsecase struct {
	mock.Mock
}

func (m *mockFirebaseLinkUsecase) LinkFirebaseUID(ctx context.Context, userID int, idToken string) error {
	args := m.Called(ctx, userID, idToken)
	return args.Error(0)
}

type mockFirebaseLoginUsecase struct {
	mock.Mock
}

func (m *mockFirebaseLoginUsecase) LoginWithFirebase(ctx context.Context, idToken string) (string, error) {
	args := m.Called(ctx, idToken)
	return args.String(0), args.Error(1)
}

type mockFirebaseRegisterUsecase struct {
	mock.Mock
}

func (m *mockFirebaseRegisterUsecase) RegisterWithFirebase(ctx context.Context, idToken string, username string) (string, error) {
	args := m.Called(ctx, idToken, username)
	return args.String(0), args.Error(1)
}

func newFirebaseHandler() (*api_internal.FirebaseHandler, *mockFirebaseLinkUsecase, *mockFirebaseLoginUsecase, *mockFirebaseRegisterUsecase) {
	linkUsecase := new(mockFirebaseLinkUsecase)
	loginUsecase := new(mockFirebaseLoginUsecase)
	registerUsecase := new(mockFirebaseRegisterUsecase)
	h := api_internal.NewFirebaseHandler(linkUsecase, loginUsecase, registerUsecase, false, http.SameSiteLaxMode)
	return h, linkUsecase, loginUsecase, registerUsecase
}

func TestFirebaseHandler_Link(t *testing.T) {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	h, linkUsecase, _, _ := newFirebaseHandler()

	t.Run("正常系: Firebase UID を連携する", func(t *testing.T) {
		linkUsecase.On("LinkFirebaseUID", mock.Anything, 1, "firebase-id-token").Return(nil).Once()

		body := `{"id_token":"firebase-id-token"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/firebase/link", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.Link(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		linkUsecase.AssertExpectations(t)
	})

	t.Run("異常系: 既に他ユーザーに紐付いている場合は409を返す", func(t *testing.T) {
		linkUsecase.On("LinkFirebaseUID", mock.Anything, 1, "firebase-id-token").Return(usecase.ErrFirebaseUIDAlreadyLinked).Once()

		body := `{"id_token":"firebase-id-token"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/firebase/link", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.Link(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeFirebaseUIDAlreadyLinked, apiErr.Code)
		linkUsecase.AssertExpectations(t)
	})
}

func TestFirebaseHandler_Login(t *testing.T) {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	h, _, loginUsecase, _ := newFirebaseHandler()

	t.Run("正常系: Firebase ログインで Cookie を発行する", func(t *testing.T) {
		loginUsecase.On("LoginWithFirebase", mock.Anything, "firebase-id-token").Return("jwt-token", nil).Once()

		body := `{"id_token":"firebase-id-token"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/firebase/login", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)

		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, "token", cookies[0].Name)
		assert.Equal(t, "jwt-token", cookies[0].Value)
		loginUsecase.AssertExpectations(t)
	})

	t.Run("異常系: 無効トークンは401を返す", func(t *testing.T) {
		loginUsecase.On("LoginWithFirebase", mock.Anything, "invalid-token").Return("", usecase.ErrInvalidIDToken).Once()

		body := `{"id_token":"invalid-token"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/firebase/login", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		loginUsecase.AssertExpectations(t)
	})
}

func TestFirebaseHandler_Register(t *testing.T) {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	h, _, _, registerUsecase := newFirebaseHandler()

	t.Run("正常系: 新規ユーザー登録で201と Cookie を返す", func(t *testing.T) {
		registerUsecase.On("RegisterWithFirebase", mock.Anything, "firebase-id-token", "newuser").Return("jwt-token", nil).Once()

		body := `{"id_token":"firebase-id-token","username":"newuser"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/firebase/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, "token", cookies[0].Name)
		assert.Equal(t, "jwt-token", cookies[0].Value)
		registerUsecase.AssertExpectations(t)
	})

	t.Run("異常系: Firebase UID が既存ユーザーに紐付いている場合は409を返す", func(t *testing.T) {
		registerUsecase.On("RegisterWithFirebase", mock.Anything, "linked-token", "newuser").Return("", usecase.ErrFirebaseUIDAlreadyLinked).Once()

		body := `{"id_token":"linked-token","username":"newuser"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/firebase/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeFirebaseUIDAlreadyLinked, apiErr.Code)
		registerUsecase.AssertExpectations(t)
	})

	t.Run("異常系: usernameバリデーション失敗で422を返す", func(t *testing.T) {
		body := `{"id_token":"firebase-id-token","username":"ab"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/firebase/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("異常系: id_tokenが未指定の場合は422を返す", func(t *testing.T) {
		body := `{"username":"newuser"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/firebase/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	})
}
