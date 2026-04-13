package apierror

import (
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
)

func TestFromUsecaseError_UID不一致系は汎用認証エラーに丸める(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "Firebase UID未連携は invalid_credentials を返す",
			err:  errors.Join(usecase.ErrInvalidCredentials, usecase.ErrFirebaseUIDNotLinked),
		},
		{
			name: "再認証UID不一致は invalid_credentials を返す",
			err:  errors.Join(usecase.ErrInvalidCredentials, usecase.ErrReauthUIDMismatch),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromUsecaseError(tt.err)

			assert.Equal(t, CodeInvalidCredentials, got.Code)
			assert.Equal(t, ErrInvalidCredentials.HTTPStatus, got.HTTPStatus)
			assert.Equal(t, tt.err, got.Internal)
		})
	}
}
