package api_internal

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockTemporaryPlayerDataUsecase struct{ mock.Mock }

const expectedTempDataMaxUncompressedBytes = 5 * 1024 * 1024

func (m *mockTemporaryPlayerDataUsecase) Create(ctx context.Context, input usecase.CreateTemporaryPlayerDataInput) (*usecase.CreateTemporaryPlayerDataOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CreateTemporaryPlayerDataOutput), args.Error(1)
}

func (m *mockTemporaryPlayerDataUsecase) Commit(ctx context.Context, input usecase.CommitTemporaryPlayerDataInput) (*playerdataresult.Result, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*playerdataresult.Result), args.Error(1)
}

func gzipJSON(t *testing.T, payload any) []byte {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return gzipBytes(t, b)
}

func gzipBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func gzipTemporaryPlayerDataJSONWithSize(t *testing.T, size int) []byte {
	t.Helper()
	payload := struct {
		Name    string `json:"name"`
		Padding string `json:"padding"`
	}{Name: "TEST"}

	emptyJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	require.GreaterOrEqual(t, size, len(emptyJSON))
	payload.Padding = strings.Repeat("a", size-len(emptyJSON))

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.Len(t, raw, size)
	return gzipBytes(t, raw)
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	mockUC := new(mockTemporaryPlayerDataUsecase)
	h := NewTemporaryPlayerDataHandler(mockUC)

	payload := usecase.PlayerDataPayload{Name: "TEST"}
	body := gzipJSON(t, payload)
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	req.RemoteAddr = "127.0.0.1:12345"

	mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateTemporaryPlayerDataInput) bool {
		return input.IPAddress == "127.0.0.1" && len(input.Payload) > 0
	})).Return(&usecase.CreateTemporaryPlayerDataOutput{UploadToken: "token", ExpiresAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}, nil).Once()

	err := h.CreateTemporaryData(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "uploadToken")
	mockUC.AssertExpectations(t)
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData_認証状態を見ない(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	mockUC := new(mockTemporaryPlayerDataUsecase)
	h := NewTemporaryPlayerDataHandler(mockUC)

	body := gzipJSON(t, map[string]any{"name": "TEST"})
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})
	req.RemoteAddr = "127.0.0.1:12345"

	mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateTemporaryPlayerDataInput) bool {
		return input.IPAddress == "127.0.0.1" && len(input.Payload) > 0
	})).Return(&usecase.CreateTemporaryPlayerDataOutput{UploadToken: "token", ExpiresAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}, nil).Once()

	err := h.CreateTemporaryData(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData_サイズ超過(t *testing.T) {
	e := echo.New()
	h := NewTemporaryPlayerDataHandler(new(mockTemporaryPlayerDataUsecase))

	tooLarge := bytes.Repeat([]byte("a"), 512001)
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(tooLarge))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateTemporaryData(c)
	require.Error(t, err)
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, http.StatusRequestEntityTooLarge, apiErr.HTTPStatus)
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData_解凍後JSONが5MiBなら受け付ける(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	mockUC := new(mockTemporaryPlayerDataUsecase)
	h := NewTemporaryPlayerDataHandler(mockUC)

	require.Equal(t, expectedTempDataMaxUncompressedBytes, info.TempDataMaxUncompressedBytes)
	body := gzipTemporaryPlayerDataJSONWithSize(t, expectedTempDataMaxUncompressedBytes)
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	req.RemoteAddr = "127.0.0.1:12345"

	mockUC.On("Create", mock.Anything, mock.Anything).Return(&usecase.CreateTemporaryPlayerDataOutput{
		UploadToken: "token",
		ExpiresAt:   time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC),
	}, nil).Once()

	err := h.CreateTemporaryData(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData_解凍後JSONが5MiBより1Byte大きい場合はサイズ超過(t *testing.T) {
	e := echo.New()
	h := NewTemporaryPlayerDataHandler(new(mockTemporaryPlayerDataUsecase))

	body := gzipTemporaryPlayerDataJSONWithSize(t, expectedTempDataMaxUncompressedBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateTemporaryData(c)
	require.Error(t, err)
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, http.StatusRequestEntityTooLarge, apiErr.HTTPStatus)
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData_ContentType不正(t *testing.T) {
	e := echo.New()
	h := NewTemporaryPlayerDataHandler(new(mockTemporaryPlayerDataUsecase))

	body := gzipJSON(t, usecase.PlayerDataPayload{Name: "TEST"})
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, "text/plain")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateTemporaryData(c)
	require.Error(t, err)
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
}

func TestTemporaryPlayerDataHandler_CommitTemporaryData(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	mockUC := new(mockTemporaryPlayerDataUsecase)
	h := NewTemporaryPlayerDataHandler(mockUC)

	reqBody := `{"uploadToken":"11111111-1111-4111-8111-111111111111"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/commit", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	mockUC.On("Commit", mock.Anything, usecase.CommitTemporaryPlayerDataInput{User: &entity.User{ID: 1}, UploadToken: "11111111-1111-4111-8111-111111111111"}).Return(&playerdataresult.Result{PlayerID: 1}, nil).Once()

	err := h.CommitTemporaryData(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestTemporaryPlayerDataHandler_CommitTemporaryData_保存済み本文が壊れている(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	mockUC := new(mockTemporaryPlayerDataUsecase)
	h := NewTemporaryPlayerDataHandler(mockUC)

	reqBody := `{"uploadToken":"11111111-1111-4111-8111-111111111111"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/commit", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	mockUC.On("Commit", mock.Anything, usecase.CommitTemporaryPlayerDataInput{
		User:        &entity.User{ID: 1},
		UploadToken: "11111111-1111-4111-8111-111111111111",
	}).Return(nil, usecase.ErrTempDataPayloadInvalidJSON).Once()

	err := h.CommitTemporaryData(c)
	require.Error(t, err)
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	mockUC.AssertExpectations(t)
}
