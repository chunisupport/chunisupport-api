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

func TestFromUsecaseError_認証失敗は汎用認証エラーに丸める(t *testing.T) {
	got := FromUsecaseError(usecase.ErrInvalidCredentials)

	assert.Equal(t, CodeInvalidCredentials, got.Code)
	assert.Equal(t, ErrInvalidCredentials.HTTPStatus, got.HTTPStatus)
	assert.Equal(t, usecase.ErrInvalidCredentials, got.Internal)
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

func TestFromUsecaseError_auth_time欠落は詳細を伏せてrecent_sign_in_requiredに丸める(t *testing.T) {
	err := errors.Join(usecase.ErrRecentSignInAuthTimeMissing, errors.New("firebase token auth_time is empty"))

	got := FromUsecaseError(err)

	assert.Equal(t, CodeRecentSignInRequired, got.Code)
	assert.Equal(t, ErrRecentSignInRequired.HTTPStatus, got.HTTPStatus)
	assert.ErrorIs(t, got.Internal, usecase.ErrRecentSignInAuthTimeMissing)
}
