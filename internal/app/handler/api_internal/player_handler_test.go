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
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockPlayerUsecase は usecase.PlayerUsecase のモックです。
type mockPlayerUsecase struct {
	mock.Mock
}

func (m *mockPlayerUsecase) CreatePlayer(ctx context.Context, userID int, name string) (*entity.Player, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func TestPlayerHandler_CreatePlayer(t *testing.T) {
	// Setup
	e := echo.New()
	e.Validator = app.NewCustomValidator()

	// モックの期待値設定
	mockUsecase := new(mockPlayerUsecase)
	name, err := playername.NewPlayerName("太郎")
	assert.NoError(t, err)
	expectedPlayer := entity.NewPlayer(1, name)
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
		assert.Equal(t, expectedPlayer.Name.String(), playerDTO.Name)
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

func TestPlayerHandler_CreatePlayer_不正JSONは400(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "Content-Typeなし", body: `{"name":"太郎"}`},
		{name: "未知フィールド", contentType: echo.MIMEApplicationJSON, body: `{"name":"太郎","unknown":1}`},
		{name: "複数JSON値", contentType: echo.MIMEApplicationJSON, body: `{"name":"太郎"} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(mockPlayerUsecase)
			h := api_internal.NewPlayerHandler(mockUsecase)
			e := echo.New()
			e.Validator = app.NewCustomValidator()
			req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}
			c := e.NewContext(req, httptest.NewRecorder())
			c.Set("userEntity", &entity.User{ID: 1})

			err := h.CreatePlayer(c)

			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
			mockUsecase.AssertNotCalled(t, "CreatePlayer", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
