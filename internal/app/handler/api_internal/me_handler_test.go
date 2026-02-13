package api_internal_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockPlayerDataUsecase は usecase.PlayerDataUsecase のモックです。
type mockPlayerDataUsecase struct {
	mock.Mock
}

func (m *mockPlayerDataUsecase) Register(ctx context.Context, user *entity.User, payload *usecase.PlayerDataPayload, hash string) (*dto_internal.PlayerDataResult, error) {
	args := m.Called(ctx, user, payload, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.PlayerDataResult), args.Error(1)
}

func (m *mockPlayerDataUsecase) Delete(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// compressAndEncodeGzipBase64 はJSONデータをgzip圧縮してbase64エンコードします。
func compressAndEncodeGzipBase64(data []byte) (string, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(data); err != nil {
		return "", err
	}
	if err := gzipWriter.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func TestMeHandler_RegisterData(t *testing.T) {
	// Setup
	e := echo.New()
	e.Validator = app.NewCustomValidator()

	mockUsecase := new(mockPlayerDataUsecase)

	// テスト用のプレイヤーデータ
	testPayload := &usecase.PlayerDataPayload{
		AppVersion: "1.0.0",
		Name:       "テストプレイヤー",
		Level:      100,
	}

	expectedResult := &dto_internal.PlayerDataResult{
		PlayerID:   1,
		AppVersion: "1.0.0",
	}

	testUser := &entity.User{
		ID:       1,
		Username: username.MustNewUserName("testuser"),
	}

	mockUsecase.On("Register", mock.Anything, testUser, mock.Anything, mock.Anything).Return(expectedResult, nil)

	h := api_internal.NewMeHandler(mockUsecase)

	t.Run("ハッピーパス: base64+gzip形式でのデータ登録", func(t *testing.T) {
		// JSONを作成
		jsonData, err := json.Marshal(testPayload)
		assert.NoError(t, err)

		// gzip圧縮 + base64エンコード
		base64Data, err := compressAndEncodeGzipBase64(jsonData)
		assert.NoError(t, err)

		// リクエスト作成
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data", bytes.NewBufferString(base64Data))
		req.Header.Set(echo.HeaderContentType, "application/octet-stream")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// ユーザー情報をコンテキストにセット
		c.Set("userEntity", testUser)

		// ハンドラ実行
		err = h.RegisterData(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// レスポンス確認
		var result dto_internal.PlayerDataResult
		err = json.Unmarshal(rec.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.PlayerID, result.PlayerID)
		assert.Equal(t, expectedResult.AppVersion, result.AppVersion)
	})

	t.Run("ハッピーパス: format=jsonでの生JSON形式データ登録", func(t *testing.T) {
		// JSONを作成
		jsonData, err := json.Marshal(testPayload)
		assert.NoError(t, err)

		// リクエスト作成（クエリパラメータ付き）
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data?format=json", bytes.NewBuffer(jsonData))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// ユーザー情報をコンテキストにセット
		c.Set("userEntity", testUser)

		// ハンドラ実行
		err = h.RegisterData(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// レスポンス確認
		var result dto_internal.PlayerDataResult
		err = json.Unmarshal(rec.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.PlayerID, result.PlayerID)
		assert.Equal(t, expectedResult.AppVersion, result.AppVersion)
	})

	t.Run("ハッピーパス: 未知のフィールドを含むJSON（エラーにならず無視される）", func(t *testing.T) {
		// 既知のフィールドと未知のフィールドを含むJSONを作成
		payloadWithUnknownFields := map[string]any{
			"app_ver":     "1.0.0",
			"name":        "テストプレイヤー",
			"level":       100,
			"extra_field": "この値は無視される",
			"debug_info":  map[string]any{"timestamp": "2024-01-01", "version": "1.0"},
			"bookmarklet": true,
		}

		jsonData, err := json.Marshal(payloadWithUnknownFields)
		assert.NoError(t, err)

		// リクエスト作成（format=json形式）
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data?format=json", bytes.NewBuffer(jsonData))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// ユーザー情報をコンテキストにセット
		c.Set("userEntity", testUser)

		// ハンドラ実行（エラーにならないことを確認）
		err = h.RegisterData(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// レスポンス確認（正常に処理されている）
		var result dto_internal.PlayerDataResult
		err = json.Unmarshal(rec.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.PlayerID, result.PlayerID)
		assert.Equal(t, expectedResult.AppVersion, result.AppVersion)
	})

	t.Run("アンハッピーパス: 認証なし", func(t *testing.T) {
		jsonData, err := json.Marshal(testPayload)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data?format=json", bytes.NewBuffer(jsonData))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// ユーザー情報をセットしない

		err = h.RegisterData(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: 空のボディ", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data?format=json", bytes.NewBufferString(""))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err := h.RegisterData(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: 不正なbase64データ", func(t *testing.T) {
		invalidBase64 := "これは不正なbase64データです!!!"
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data", bytes.NewBufferString(invalidBase64))
		req.Header.Set(echo.HeaderContentType, "application/octet-stream")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err := h.RegisterData(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: 不正なgzip圧縮データ", func(t *testing.T) {
		// 正しくないgzipデータをbase64エンコード
		invalidGzipData := base64.StdEncoding.EncodeToString([]byte("これはgzipではありません"))
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data", bytes.NewBufferString(invalidGzipData))
		req.Header.Set(echo.HeaderContentType, "application/octet-stream")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err := h.RegisterData(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: 不正なJSON（format=json）", func(t *testing.T) {
		invalidJSON := `{"invalid_json": }`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data?format=json", bytes.NewBufferString(invalidJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err := h.RegisterData(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: 不正なJSON（base64+gzip）", func(t *testing.T) {
		// 不正なJSONをgzip圧縮してbase64エンコード
		invalidJSON := `{"invalid_json": }`
		base64Data, err := compressAndEncodeGzipBase64([]byte(invalidJSON))
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data", bytes.NewBufferString(base64Data))
		req.Header.Set(echo.HeaderContentType, "application/octet-stream")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err = h.RegisterData(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("アンハッピーパス: usecaseエラー", func(t *testing.T) {
		mockUsecaseWithError := new(mockPlayerDataUsecase)
		mockUsecaseWithError.On("Register", mock.Anything, testUser, mock.Anything, mock.Anything).Return(nil, errors.New("usecase error"))

		hWithError := api_internal.NewMeHandler(mockUsecaseWithError)

		jsonData, err := json.Marshal(testPayload)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/internal/me/register-data?format=json", bytes.NewBuffer(jsonData))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err = hWithError.RegisterData(c)
		assert.Error(t, err)
	})

	t.Run("ハッピーパス: プレイヤーデータ削除", func(t *testing.T) {
		mockUsecaseWithDelete := new(mockPlayerDataUsecase)
		mockUsecaseWithDelete.On("Delete", mock.Anything, testUser).Return(nil)

		deleteHandler := api_internal.NewMeHandler(mockUsecaseWithDelete)

		req := httptest.NewRequest(http.MethodDelete, "/internal/me/player-data", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", testUser)

		err := deleteHandler.DeletePlayerData(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("アンハッピーパス: 認証なし", func(t *testing.T) {
		mockUsecaseWithDelete := new(mockPlayerDataUsecase)
		deleteHandler := api_internal.NewMeHandler(mockUsecaseWithDelete)

		req := httptest.NewRequest(http.MethodDelete, "/internal/me/player-data", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := deleteHandler.DeletePlayerData(c)
		assert.Error(t, err)
		assert.Equal(t, apierror.ErrUnauthorized, err)
	})
}
