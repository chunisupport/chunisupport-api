package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dataTransferTestUserID はテスト用の認証ユーザーIDです。
// 0（未設定）や1のような取り違えに気づきにくい値を避けた固定値にしています。
const dataTransferTestUserID = 424242

// dataTransferBodyLimitBytes は移行ファイル本文の上限です。
// 仕様（docs/user_data_transfer_specification.md の6.3節、docs/API.md）で32MiBと定められているための固定値です。
const dataTransferBodyLimitBytes = 33554432

// 上限付近の検証には32MiB級の本文確保が必要なため、ValidateとImportのテストで同じバッファを使い回します。
var (
	dataTransferLimitBody = bytes.Repeat([]byte("a"), dataTransferBodyLimitBytes)
	// 上限+1バイトの本文です。長さの期待値は仕様から求めた固定値33554433です。
	dataTransferOverLimitBody = bytes.Repeat([]byte("a"), 33554433)
)

type dataTransferUsecaseStub struct {
	exportOutput   *usecase.UserDataTransferExportOutput
	validateOutput *usecase.UserDataTransferValidationOutput
	importOutput   *usecase.UserDataTransferImportOutput

	exportCalled   bool
	exportUserID   int
	validateCalled bool
	validateUserID int
	validateBody   []byte
	importCalled   bool
	importUserID   int
	importBody     []byte
}

func (s *dataTransferUsecaseStub) Export(_ context.Context, userID int) (*usecase.UserDataTransferExportOutput, error) {
	// ハンドラーが認証ユーザーのIDを正しく渡すことを検証するため、呼び出しの有無と引数を記録します。
	s.exportCalled = true
	s.exportUserID = userID
	return s.exportOutput, nil
}

func (s *dataTransferUsecaseStub) Validate(_ context.Context, userID int, encoded []byte) (*usecase.UserDataTransferValidationOutput, error) {
	// ハンドラーが受信した本文をそのまま渡すことを検証するため、呼び出しの有無と引数を記録します。
	s.validateCalled = true
	s.validateUserID = userID
	s.validateBody = encoded
	return s.validateOutput, nil
}

func (s *dataTransferUsecaseStub) Import(_ context.Context, userID int, encoded []byte) (*usecase.UserDataTransferImportOutput, error) {
	// ハンドラーが受信した本文をそのまま渡すことを検証するため、呼び出しの有無と引数を記録します。
	s.importCalled = true
	s.importUserID = userID
	s.importBody = encoded
	return s.importOutput, nil
}

func TestDataTransferHandlerExportReturnsAttachment(t *testing.T) {
	// Given: 認証ユーザーに紐づく移行ファイルが用意できている
	stub := &dataTransferUsecaseStub{exportOutput: &usecase.UserDataTransferExportOutput{File: []byte("{\"signed\":true}")}}
	handler := NewDataTransferHandler(stub)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/export", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

	// When: エクスポートを実行する
	err := handler.Export(ctx)

	// Then: 認証ユーザーのファイルが添付として返り、usecaseへ認証ユーザーのIDが渡る
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, "attachment; filename=\"chunisupport-transfer.json\"", rec.Header().Get(echo.HeaderContentDisposition))
	assert.JSONEq(t, "{\"signed\":true}", rec.Body.String())
	assert.True(t, stub.exportCalled)
	assert.Equal(t, dataTransferTestUserID, stub.exportUserID)
}

func TestDataTransferHandlerValidatePassesEnvelope(t *testing.T) {
	// Given: 検証可能な移行ファイル本文が送られている
	body := []byte("{\"signed\":true}")
	output := &usecase.UserDataTransferValidationOutput{
		Importable:           true,
		PlayerName:           "テスト",
		Blockers:             []string{},
		UnresolvedReferences: []string{},
	}
	stub := &dataTransferUsecaseStub{validateOutput: output}
	handler := NewDataTransferHandler(stub)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

	// When: 検証を実行する
	err := handler.Validate(ctx)

	// Then: 受信した本文がそのままusecaseへ渡り、検証結果が返る
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response dto_internal.DataTransferValidationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Importable)
	assert.True(t, stub.validateCalled)
	assert.Equal(t, dataTransferTestUserID, stub.validateUserID)
	assert.Equal(t, body, stub.validateBody)
}

func TestDataTransferHandlerValidateReturnsEmptyArraysForNoValidationIssues(t *testing.T) {
	// Given: 問題のない検証結果が返る状態
	body := []byte("{\"signed\":true}")
	output := &usecase.UserDataTransferValidationOutput{
		Importable: true,
		PlayerName: "TEST",
	}
	stub := &dataTransferUsecaseStub{validateOutput: output}
	handler := NewDataTransferHandler(stub)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

	// When: 検証を実行する
	err := handler.Validate(ctx)

	// Then: 空配列を含む応答が返り、受信した本文がそのままusecaseへ渡る
	require.NoError(t, err)
	assert.JSONEq(t, `{"importable":true,"player_name":"TEST","counts":{"records":0,"record_histories":0,"worldsend_records":0,"worldsend_record_histories":0,"metric_histories":0,"course_records":0,"honors":0,"favorite_songs":0,"locked_songs":0,"goal_groups":0,"goals":0,"record_filters":0},"blockers":[],"unresolved_references":[],"unresolved_reference_count":0}`, rec.Body.String())
	assert.True(t, stub.validateCalled)
	assert.Equal(t, dataTransferTestUserID, stub.validateUserID)
	assert.Equal(t, body, stub.validateBody)
}

func TestDataTransferHandlerValidateRejectsEmptyBodyAsInvalidFile(t *testing.T) {
	handler := NewDataTransferHandler(&dataTransferUsecaseStub{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

	err := handler.Validate(ctx)

	assert.Same(t, apierror.ErrDataTransferInvalidFile, err)
}

func TestDataTransferHandlerImportReturnsImportedPlayer(t *testing.T) {
	// Given: 取り込み可能な移行ファイル本文が送られている
	body := []byte("{\"signed\":true}")
	output := &usecase.UserDataTransferImportOutput{
		PlayerID: 42,
		Counts:   usecase.UserDataTransferCounts{Records: 3, Goals: 2},
	}
	stub := &dataTransferUsecaseStub{importOutput: output}
	handler := NewDataTransferHandler(stub)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/import", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

	// When: 取り込みを実行する
	err := handler.Import(ctx)

	// Then: 新しいプレイヤーIDと件数が返り、受信した本文がそのままusecaseへ渡る
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response dto_internal.DataTransferImportResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, 42, response.PlayerID)
	assert.Equal(t, 3, response.Counts.Records)
	assert.Equal(t, 2, response.Counts.Goals)
	assert.True(t, stub.importCalled)
	assert.Equal(t, dataTransferTestUserID, stub.importUserID)
	assert.Equal(t, body, stub.importBody)
}

func TestDataTransferHandlerValidateBodySizeLimits(t *testing.T) {
	tests := []struct {
		name string
		// Given: 前提条件
		body []byte
		// Then: 期待する結果
		wantCalled bool
		wantLength int
	}{
		{
			name:       "1バイトの本文はそのままusecaseへ渡る",
			body:       []byte("a"),
			wantCalled: true,
			wantLength: 1,
		},
		{
			name:       "上限ちょうどの本文はそのままusecaseへ渡る",
			body:       dataTransferLimitBody,
			wantCalled: true,
			wantLength: 33554432,
		},
		{
			name:       "上限+1バイトの本文はusecaseを呼ばずに拒否する",
			body:       dataTransferOverLimitBody,
			wantCalled: false,
			wantLength: 33554433,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 任意のバイト列からなる本文が送られている。本文の形式検証はスタブ側ではなくusecase側の責務のため、内容は問いません。
			stub := &dataTransferUsecaseStub{validateOutput: &usecase.UserDataTransferValidationOutput{Importable: true}}
			handler := NewDataTransferHandler(stub)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", bytes.NewReader(tt.body))
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

			// When: 検証を実行する
			err := handler.Validate(ctx)

			// Then: 受理時は本文がそのままusecaseへ渡り、拒否時は413で止まりusecaseを呼ばない
			if tt.wantCalled {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.True(t, stub.validateCalled)
				assert.Equal(t, dataTransferTestUserID, stub.validateUserID)
				assert.Len(t, stub.validateBody, tt.wantLength)
				assert.True(t, bytes.Equal(tt.body, stub.validateBody), "受信した本文と同じ内容がusecaseへ渡ること")
			} else {
				assert.Same(t, apierror.ErrPayloadTooLarge, err)
				apiErr, ok := err.(*apierror.APIError)
				require.True(t, ok)
				assert.Equal(t, 413, apiErr.HTTPStatus)
				assert.Equal(t, "payload_too_large", apiErr.Code)
				assert.False(t, stub.validateCalled)
			}
		})
	}
}

func TestDataTransferHandlerImportBodySizeLimits(t *testing.T) {
	tests := []struct {
		name string
		// Given: 前提条件
		body []byte
		// Then: 期待する結果
		wantCalled bool
		wantLength int
	}{
		{
			name:       "1バイトの本文はそのままusecaseへ渡る",
			body:       []byte("a"),
			wantCalled: true,
			wantLength: 1,
		},
		{
			name:       "上限ちょうどの本文はそのままusecaseへ渡る",
			body:       dataTransferLimitBody,
			wantCalled: true,
			wantLength: 33554432,
		},
		{
			name:       "上限+1バイトの本文はusecaseを呼ばずに拒否する",
			body:       dataTransferOverLimitBody,
			wantCalled: false,
			wantLength: 33554433,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 任意のバイト列からなる本文が送られている。本文の形式検証はスタブ側ではなくusecase側の責務のため、内容は問いません。
			stub := &dataTransferUsecaseStub{importOutput: &usecase.UserDataTransferImportOutput{PlayerID: 42}}
			handler := NewDataTransferHandler(stub)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/import", bytes.NewReader(tt.body))
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.Set("userEntity", &entity.User{ID: dataTransferTestUserID})

			// When: 取り込みを実行する
			err := handler.Import(ctx)

			// Then: 受理時は本文がそのままusecaseへ渡り、拒否時は413で止まりusecaseを呼ばない
			if tt.wantCalled {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.True(t, stub.importCalled)
				assert.Equal(t, dataTransferTestUserID, stub.importUserID)
				assert.Len(t, stub.importBody, tt.wantLength)
				assert.True(t, bytes.Equal(tt.body, stub.importBody), "受信した本文と同じ内容がusecaseへ渡ること")
			} else {
				assert.Same(t, apierror.ErrPayloadTooLarge, err)
				apiErr, ok := err.(*apierror.APIError)
				require.True(t, ok)
				assert.Equal(t, 413, apiErr.HTTPStatus)
				assert.Equal(t, "payload_too_large", apiErr.Code)
				assert.False(t, stub.importCalled)
			}
		})
	}
}
