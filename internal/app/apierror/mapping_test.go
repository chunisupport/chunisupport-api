package apierror

import (
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
)

func TestFromUsecaseError_認証失敗は汎用認証エラーに丸める(t *testing.T) {
	got := FromUsecaseError(usecase.ErrInvalidCredentials)

	assert.Equal(t, CodeInvalidCredentials, got.Code)
	assert.Equal(t, ErrInvalidCredentials.HTTPStatus, got.HTTPStatus)
	assert.Equal(t, usecase.ErrInvalidCredentials, got.Internal)
}

func TestFromUsecaseError_auth_time欠落は詳細を伏せてrecent_sign_in_requiredに丸める(t *testing.T) {
	err := errors.Join(usecase.ErrRecentSignInAuthTimeMissing, errors.New("firebase token auth_time is empty"))

	got := FromUsecaseError(err)

	assert.Equal(t, CodeRecentSignInRequired, got.Code)
	assert.Equal(t, ErrRecentSignInRequired.HTTPStatus, got.HTTPStatus)
	assert.ErrorIs(t, got.Internal, usecase.ErrRecentSignInAuthTimeMissing)
}
