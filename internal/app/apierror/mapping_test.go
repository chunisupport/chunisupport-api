package apierror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromUsecaseError_スコア履歴時刻競合を409へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(repository.ErrScoreHistoryTimestampConflict)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeConflict, apiErr.Code)
}

func TestFromUsecaseError_公式指標履歴時刻競合を409へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(repository.ErrPlayerMetricHistoryTimestampConflict)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeConflict, apiErr.Code)
}

func TestFromUsecaseError_公式指標履歴なしを404へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrPlayerMetricHistoryNotFound)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
	assert.Equal(t, CodePlayerMetricHistoryNotFound, apiErr.Code)
}

func TestFromUsecaseError_プレイヤーデータ検証エラーを422へ変換する(t *testing.T) {
	validationErr := &usecase.PlayerDataValidationError{Field: "updated_at", Message: "must be RFC3339"}

	apiErr := FromUsecaseError(validationErr)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, CodeValidationFailed, apiErr.Code)
	assert.ErrorIs(t, apiErr, validationErr)
}

func TestFromUsecaseError_認証失敗は汎用認証エラーに丸める(t *testing.T) {
	got := FromUsecaseError(usecase.ErrInvalidCredentials)

	assert.Equal(t, CodeInvalidCredentials, got.Code)
	assert.Equal(t, ErrInvalidCredentials.HTTPStatus, got.HTTPStatus)
	assert.Equal(t, usecase.ErrInvalidCredentials, got.Internal)
}

func TestFromUsecaseError_メンテナンス中は専用503へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrMaintenanceMode)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.HTTPStatus)
	assert.Equal(t, CodeMaintenanceMode, apiErr.Code)
	assert.ErrorIs(t, apiErr.Internal, usecase.ErrMaintenanceMode)
}

func TestFromUsecaseError_不正なメンテナンスコメントは400へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidMaintenanceComment)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeBadRequest, apiErr.Code)
	assert.ErrorIs(t, apiErr.Internal, usecase.ErrInvalidMaintenanceComment)
}

func TestFromUsecaseError_禁止ユーザー名は登録失敗に丸める(t *testing.T) {
	got := FromUsecaseError(usecase.ErrUsernameForbidden)

	assert.Equal(t, CodeRegistrationFailed, got.Code)
	assert.Equal(t, http.StatusBadRequest, got.HTTPStatus)
}

func TestFromUsecaseError_お気に入り上限エラーを400に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrPlayerFavoriteSongLimitExceeded)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeFavoriteSongLimitExceeded, apiErr.Code)
}

func TestFromUsecaseError_目標並び順不正を400に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidGoalOrder)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeGoalInvalidOrder, apiErr.Code)
}

func TestFromUsecaseError_目標グループ名重複を409に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrGoalGroupConflict)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeGoalGroupConflict, apiErr.Code)
}

func TestFromUsecaseError_auth_time欠落は詳細を伏せてrecent_sign_in_requiredに丸める(t *testing.T) {
	err := errors.Join(usecase.ErrRecentSignInAuthTimeMissing, errors.New("firebase token auth_time is empty"))

	got := FromUsecaseError(err)

	assert.Equal(t, CodeRecentSignInRequired, got.Code)
	assert.Equal(t, ErrRecentSignInRequired.HTTPStatus, got.HTTPStatus)
	assert.ErrorIs(t, got.Internal, usecase.ErrRecentSignInAuthTimeMissing)
}

func TestFromUsecaseError_不正なコース入力をバリデーションエラーへ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidCourseInput)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, CodeValidationFailed, apiErr.Code)
}

func TestFromUsecaseError_コース未検出を404へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrCourseNotFound)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
}

func TestFromUsecaseError_APIトークン管理エラーを変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "ID不正", err: usecase.ErrInvalidAPITokenID, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAPITokenID},
		{name: "名前不正", err: usecase.ErrInvalidAPITokenName, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAPITokenName},
		{name: "上限超過", err: usecase.ErrAPITokenLimitExceeded, wantStatus: http.StatusBadRequest, wantCode: CodeAPITokenLimitExceeded},
		{name: "名前重複", err: usecase.ErrAPITokenNameConflict, wantStatus: http.StatusConflict, wantCode: CodeAPITokenNameConflict},
		{name: "未検出", err: usecase.ErrAPITokenNotFound, wantStatus: http.StatusNotFound, wantCode: CodeAPITokenNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := FromUsecaseError(tt.err)

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}

func TestFromUsecaseError_データ移行エラーを専用HTTPエラーへ変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantStatus int
	}{
		{name: "プレイヤーなし", err: usecase.ErrDataTransferPlayerNotFound, wantCode: CodeDataTransferPlayerNotFound, wantStatus: http.StatusBadRequest},
		{name: "ファイル不正", err: usecase.ErrDataTransferInvalidFile, wantCode: CodeDataTransferInvalidFile, wantStatus: http.StatusBadRequest},
		{name: "署名不正", err: usecase.ErrDataTransferInvalidSignature, wantCode: CodeDataTransferInvalidSignature, wantStatus: http.StatusBadRequest},
		{name: "スキーマ非対応", err: usecase.ErrDataTransferUnsupportedSchema, wantCode: CodeDataTransferUnsupportedSchema, wantStatus: http.StatusBadRequest},
		{name: "データ不正", err: usecase.ErrDataTransferInvalidData, wantCode: CodeDataTransferInvalidData, wantStatus: http.StatusBadRequest},
		{name: "参照未解決", err: usecase.ErrDataTransferUnresolvedReference, wantCode: CodeDataTransferUnresolvedReference, wantStatus: http.StatusBadRequest},
		{name: "移行先に既存データあり", err: usecase.ErrDataTransferDestinationNotEmpty, wantCode: CodeDataTransferDestinationNotEmpty, wantStatus: http.StatusConflict},
		{name: "サイズ超過", err: usecase.ErrDataTransferPayloadTooLarge, wantCode: CodePayloadTooLarge, wantStatus: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := FromUsecaseError(tt.err)

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.ErrorIs(t, apiErr, tt.err)
		})
	}
}
