package api_internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockPlayerUsecase は usecase.PlayerUsecase のモックです。
type mockPlayerUsecase struct {
	mock.Mock
}

func (m *mockPlayerUsecase) CreatePlayer(ctx context.Context, userID int, name string) (*dto.PlayerDTO, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PlayerDTO), args.Error(1)
}

func TestPlayerHandler_CreatePlayer(t *testing.T) {
	// Setup
	e := echo.New()
	e.Validator = app.NewCustomValidator()

	// モックの期待値設定
	mockUsecase := new(mockPlayerUsecase)
	expectedPlayer := &dto.PlayerDTO{Name: "太郎"}
	mockUsecase.On("CreatePlayer", mock.Anything, 1, "太郎").Return(expectedPlayer, nil)
	mockUsecase.On("CreatePlayer", mock.Anything, 1, "エラープレイヤー").Return(nil, errors.New("failed to create player"))
	mockUsecase.On("CreatePlayer", mock.Anything, 1, "不正名").Return(nil, usecase.ErrInvalidPlayerName)

	h := api_internal.NewPlayerHandler(mockUsecase)

	t.Run("ハッピーパス: 正常なプレイヤー作成", func(t *testing.T) {
		body := `{"name": "太郎"}`
		req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.CreatePlayer(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var playerDTO dto.PlayerDTO
		err = json.Unmarshal(rec.Body.Bytes(), &playerDTO)
		assert.NoError(t, err)
		assert.Equal(t, expectedPlayer.Name, playerDTO.Name)
	})

	t.Run("アンハッピーパス: 入力検証エラー（名前が長すぎる）", func(t *testing.T) {
		longName := string(bytes.Repeat([]byte("a"), 51)) // 51文字の'a'
		body := `{"name": "` + longName + `"}`
		req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.CreatePlayer(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("アンハッピーパス: サービスエラー", func(t *testing.T) {
		body := `{"name": "エラープレイヤー"}`
		req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.CreatePlayer(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: プレイヤー名バリデーションエラー", func(t *testing.T) {
		body := `{"name": "不正名"}`
		req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.CreatePlayer(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})
	t.Run("アンハッピーパス: 未認証（userEntityなし）", func(t *testing.T) {
		body := `{"name": "太郎"}`
		req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.CreatePlayer(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
	})
}
