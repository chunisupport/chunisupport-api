package usecase

import "errors"

var (
	ErrUsernameTaken      = errors.New("this username is already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserIDMismatch     = errors.New("user ID mismatch")
	ErrInvalidSession     = errors.New("session invalid or expired")
	ErrUserDeleted        = errors.New("user deleted")

	ErrUsernameEmpty       = errors.New("username cannot be empty")
	ErrUsernameTooShort    = errors.New("username must be at least 5 characters")
	ErrUsernameTooLong     = errors.New("username must be 50 characters or less")
	ErrUsernameInvalidChar = errors.New("username can only contain lowercase letters and numbers")

	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must be 128 characters or less")
)
