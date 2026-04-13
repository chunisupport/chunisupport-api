package apierror

import (
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
