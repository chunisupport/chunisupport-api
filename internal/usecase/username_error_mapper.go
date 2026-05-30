package usecase

import (
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

func convertUsernameError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, username.ErrEmpty):
		return ErrUsernameEmpty
	case errors.Is(err, username.ErrTooShort):
		return ErrUsernameTooShort
	case errors.Is(err, username.ErrTooLong):
		return ErrUsernameTooLong
	case errors.Is(err, username.ErrInvalidChar):
		return ErrUsernameInvalidChar
	default:
		return err
	}
}
