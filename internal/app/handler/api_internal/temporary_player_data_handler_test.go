package api_internal

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
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

func newTemporaryDataRequest(body []byte, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = remoteAddr
	return req
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

func TestTemporaryPlayerDataHandler_CreateTemporaryData_IPv6は64単位で保持件数を数える(t *testing.T) {
	// Given
	e := echo.New()
	mockUC := new(mockTemporaryPlayerDataUsecase)
	h := NewTemporaryPlayerDataHandler(mockUC)

	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/temp", bytes.NewReader(gzipJSON(t, map[string]any{"name": "TEST"})))
	req.Header.Set(echo.HeaderContentEncoding, "gzip")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "[2001:db8:1:2:3:4:5:6]:12345"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateTemporaryPlayerDataInput) bool {
		return input.IPAddress == "2001:db8:1:2::/64"
	})).Return(&usecase.CreateTemporaryPlayerDataOutput{UploadToken: "token", ExpiresAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}, nil).Once()

	// When
	err := h.CreateTemporaryData(c)

	// Then
	require.NoError(t, err)
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

func TestTemporaryPlayerDataHandler_CreateTemporaryData_受信したgzip本文をそのまま渡す(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "展開後に膨らむ空要素の配列",
			json: `{"scores":{"standard":[` + strings.TrimSuffix(strings.Repeat("{},", 1000), ",") + `]}}`,
		},
		{
			name: "JSONとして壊れた本文",
			json: `{"name":"TEST"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			mockUC := new(mockTemporaryPlayerDataUsecase)
			h := NewTemporaryPlayerDataHandler(mockUC)
			body := gzipBytes(t, []byte(tt.json))
			hash := sha256.Sum256([]byte(tt.json))
			req := newTemporaryDataRequest(body, "192.0.2.1:12345")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateTemporaryPlayerDataInput) bool {
				return bytes.Equal(input.Payload, body) && input.BodyHash == hex.EncodeToString(hash[:])
			})).Return(&usecase.CreateTemporaryPlayerDataOutput{UploadToken: "token", ExpiresAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}, nil).Once()

			// When
			err := h.CreateTemporaryData(c)

			// Then
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestTemporaryPlayerDataHandler_CreateTemporaryData_gzipとして壊れた本文は400(t *testing.T) {
	valid := gzipBytes(t, []byte(`{"name":"TEST"}`))
	tests := []struct {
		name string
		body []byte
	}{
		{name: "gzipではない本文", body: []byte(`{"name":"TEST"}`)},
		{name: "途中で切れたgzip", body: valid[:len(valid)-4]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			h := NewTemporaryPlayerDataHandler(new(mockTemporaryPlayerDataUsecase))
			c := e.NewContext(newTemporaryDataRequest(tt.body, "192.0.2.1:12345"), httptest.NewRecorder())

			// When
			err := h.CreateTemporaryData(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		})
	}
}

// newRealTemporaryPlayerDataHandler は本番と同じ上限の実リポジトリにつないだハンドラを返します。
func newRealTemporaryPlayerDataHandler(playerDataUsecase usecase.PlayerDataUsecase) *TemporaryPlayerDataHandler {
	repo := infrarepo.NewTemporaryPlayerDataRepository(info.TempDataMaxEntriesPerIP, info.DefaultTempDataMaxTotalMB*1024*1024)
	return NewTemporaryPlayerDataHandler(usecase.NewTemporaryPlayerDataUsecase(nil, repo, playerDataUsecase, info.TempDataTTL, DecodeTemporaryPlayerDataPayload))
}

func TestTemporaryPlayerData_展開後に膨らむ本文でも全体容量を消費しない(t *testing.T) {
	// Given: 再シリアライズすると全体容量を超えるサイズに膨らむ本文
	e := echo.New()
	h := newRealTemporaryPlayerDataHandler(nil)
	bomb := `{"scores":{"standard":[` + strings.TrimSuffix(strings.Repeat("{},", 638000), ",") + `]}}`
	entries := []string{"192.0.2.1:12345", "192.0.2.1:12345", "192.0.2.1:12345"}

	// When
	for _, remoteAddr := range entries {
		rec := httptest.NewRecorder()
		require.NoError(t, h.CreateTemporaryData(e.NewContext(newTemporaryDataRequest(gzipBytes(t, []byte(bomb)), remoteAddr), rec)))
		require.Equal(t, http.StatusCreated, rec.Code)
	}

	// Then
	rec := httptest.NewRecorder()
	require.NoError(t, h.CreateTemporaryData(e.NewContext(newTemporaryDataRequest(gzipBytes(t, []byte(`{"name":"TEST"}`)), "198.51.100.1:12345"), rec)))
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestTemporaryPlayerData_少数の送信元では全体容量を枯渇させられない(t *testing.T) {
	// Given: 圧縮が効かず、1件あたりの上限に近い本文
	e := echo.New()
	h := newRealTemporaryPlayerDataHandler(nil)
	random := make([]byte, info.TempDataMaxCompressedBytes-1024)
	_, err := rand.Read(random)
	require.NoError(t, err)
	body := gzipBytes(t, random)
	require.LessOrEqual(t, len(body), info.TempDataMaxCompressedBytes)

	const minSourcesToExhaust = 40
	maxBytesPerSource := info.TempDataMaxEntriesPerIP * info.TempDataMaxCompressedBytes
	assert.GreaterOrEqual(t, info.DefaultTempDataMaxTotalMB*1024*1024/maxBytesPerSource, minSourcesToExhaust)

	// When: 複数の /64 がそれぞれ上限まで保存する
	for source := range minSourcesToExhaust - 1 {
		for host := range info.TempDataMaxEntriesPerIP {
			remoteAddr := fmt.Sprintf("[2001:db8:%x::%x]:12345", source, host+1)
			rec := httptest.NewRecorder()
			require.NoError(t, h.CreateTemporaryData(e.NewContext(newTemporaryDataRequest(body, remoteAddr), rec)))
			require.Equal(t, http.StatusCreated, rec.Code)
		}

		// Then: 同じ /64 からは別アドレスでも上限を超えて保存できない
		err := h.CreateTemporaryData(e.NewContext(newTemporaryDataRequest(body, fmt.Sprintf("[2001:db8:%x::ffff]:12345", source)), httptest.NewRecorder()))
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	}

	// Then: 他の送信元は保存できる
	rec := httptest.NewRecorder()
	require.NoError(t, h.CreateTemporaryData(e.NewContext(newTemporaryDataRequest(body, "198.51.100.1:12345"), rec)))
	assert.Equal(t, http.StatusCreated, rec.Code)
}

type capturePlayerDataUsecase struct {
	usecase.PlayerDataUsecase
	payload *usecase.PlayerDataPayload
	hash    string
}

func (s *capturePlayerDataUsecase) Register(_ context.Context, _ *entity.User, payload *usecase.PlayerDataPayload, hash string) (*playerdataresult.Result, error) {
	s.payload = payload
	s.hash = hash
	return &playerdataresult.Result{PlayerID: 1}, nil
}

func TestTemporaryPlayerData_一時保存した本文をcommit時に登録できる(t *testing.T) {
	// Given
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	captured := &capturePlayerDataUsecase{}
	h := newRealTemporaryPlayerDataHandler(captured)
	raw := []byte(`{"app_ver":"1.0","name":"TEST","scores":{"standard":[{"diff":"MASTER","idx":"1","score":1000000}]},"unknown":"ignored"}`)
	hash := sha256.Sum256(raw)

	createRec := httptest.NewRecorder()
	require.NoError(t, h.CreateTemporaryData(e.NewContext(newTemporaryDataRequest(gzipBytes(t, raw), "192.0.2.1:12345"), createRec)))
	var created createTemporaryPlayerDataResponse
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))

	req := httptest.NewRequest(http.MethodPost, "/internal/player-data/commit", strings.NewReader(`{"uploadToken":"`+created.UploadToken+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	commitRec := httptest.NewRecorder()
	c := e.NewContext(req, commitRec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := h.CommitTemporaryData(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, commitRec.Code)
	require.NotNil(t, captured.payload)
	assert.Equal(t, "1.0", captured.payload.AppVersion)
	assert.Equal(t, "TEST", captured.payload.Name)
	require.Len(t, captured.payload.Scores.Standard, 1)
	assert.Equal(t, "MASTER", captured.payload.Scores.Standard[0].Diff)
	assert.Equal(t, hex.EncodeToString(hash[:]), captured.hash)
}

func TestDecodeTemporaryPlayerDataPayload_不正な保存本文はエラー(t *testing.T) {
	tests := []struct {
		name   string
		stored []byte
	}{
		{name: "JSONとして壊れている", stored: gzipBytes(t, []byte(`{"name":"TEST"`))},
		{name: "gzipではない", stored: []byte(`{"name":"TEST"}`)},
		{name: "展開後サイズが上限を超える", stored: gzipTemporaryPlayerDataJSONWithSize(t, expectedTempDataMaxUncompressedBytes+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := DecodeTemporaryPlayerDataPayload(tt.stored)

			// Then
			assert.Error(t, err)
		})
	}
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

func TestTemporaryPlayerDataHandler_CommitTemporaryData_処理中のトークンは409(t *testing.T) {
	// Given
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
	}).Return(nil, usecase.ErrTemporaryPlayerDataInProgress).Once()

	// When
	err := h.CommitTemporaryData(c)

	// Then
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	mockUC.AssertExpectations(t)
}
