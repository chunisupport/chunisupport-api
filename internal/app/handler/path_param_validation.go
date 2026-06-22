package handler

import (
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// ValidateDisplayID は境界で受け取った楽曲DisplayIDを検証します。
// Usecaseへ渡す前にVOと同じ形式制約を適用し、無効な入力を早期に拒否します。
func ValidateDisplayID(value string) (string, *apierror.APIError) {
	if _, err := displayid.NewDisplayID(value); err != nil {
		return "", apierror.ErrValidationFailed.WithInternal(err)
	}
	return value, nil
}

// ValidateUsername は境界で受け取ったユーザー名を検証します。
// ユーザー登録時と同じVO制約を適用し、既存の詳細エラーコードへ変換します。
func ValidateUsername(value string) (string, *apierror.APIError) {
	if _, err := username.NewUserName(value); err != nil {
		return "", usernameValidationAPIError(err)
	}
	return value, nil
}

func usernameValidationAPIError(err error) *apierror.APIError {
	switch {
	case errors.Is(err, username.ErrEmpty):
		return apierror.ErrUsernameEmpty.WithInternal(err)
	case errors.Is(err, username.ErrTooShort):
		return apierror.ErrUsernameTooShort.WithInternal(err)
	case errors.Is(err, username.ErrTooLong):
		return apierror.ErrUsernameTooLong.WithInternal(err)
	case errors.Is(err, username.ErrInvalidChar):
		return apierror.ErrUsernameInvalidChar.WithInternal(err)
	default:
		return apierror.ErrValidationFailed.WithInternal(err)
	}
}
